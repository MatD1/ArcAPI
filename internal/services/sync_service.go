package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
	"github.com/robfig/cron/v3"
)

type SyncService struct {
	questRepo         *repository.QuestRepository
	itemRepo          *repository.ItemRepository
	skillNodeRepo     *repository.SkillNodeRepository
	hideoutModuleRepo *repository.HideoutModuleRepository
	botRepo           *repository.BotRepository
	mapRepo           *repository.MapRepository
	traderRepo        *repository.TraderRepository
	projectRepo       *repository.ProjectRepository
	dataCacheService  *DataCacheService
	githubClient      *github.Client
	cfg               *config.Config
	cron              *cron.Cron
	mu                sync.Mutex
	isRunning         bool
}

func NewSyncService(
	questRepo *repository.QuestRepository,
	itemRepo *repository.ItemRepository,
	skillNodeRepo *repository.SkillNodeRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
	botRepo *repository.BotRepository,
	mapRepo *repository.MapRepository,
	traderRepo *repository.TraderRepository,
	projectRepo *repository.ProjectRepository,
	cfg *config.Config,
) *SyncService {
	return NewSyncServiceWithCache(questRepo, itemRepo, skillNodeRepo, hideoutModuleRepo, botRepo, mapRepo, traderRepo, projectRepo, nil, cfg)
}

func NewSyncServiceWithCache(
	questRepo *repository.QuestRepository,
	itemRepo *repository.ItemRepository,
	skillNodeRepo *repository.SkillNodeRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
	botRepo *repository.BotRepository,
	mapRepo *repository.MapRepository,
	traderRepo *repository.TraderRepository,
	projectRepo *repository.ProjectRepository,
	dataCacheService *DataCacheService,
	cfg *config.Config,
) *SyncService {
	// GitHub client without auth for public repo but route through a rate-limit aware transport.
	client := github.NewClient(newRateLimitAwareHTTPClient())

	service := &SyncService{
		questRepo:         questRepo,
		itemRepo:          itemRepo,
		skillNodeRepo:     skillNodeRepo,
		hideoutModuleRepo: hideoutModuleRepo,
		botRepo:           botRepo,
		mapRepo:           mapRepo,
		traderRepo:        traderRepo,
		projectRepo:       projectRepo,
		dataCacheService:  dataCacheService,
		githubClient:      client,
		cfg:               cfg,
		cron:              cron.New(),
	}

	return service
}

// NewSyncServiceWithMissionRepo is deprecated, use NewSyncService instead
// This function is kept for backward compatibility but should not be used
func NewSyncServiceWithMissionRepo(
	missionRepo *repository.MissionRepository,
	itemRepo *repository.ItemRepository,
	skillNodeRepo *repository.SkillNodeRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
	cfg *config.Config,
) *SyncService {
	// MissionRepository is just an alias for QuestRepository
	// We need to get the underlying DB to create a new QuestRepository
	// For now, return nil as this is deprecated
	// In practice, callers should use NewSyncService directly
	return nil
}

func (s *SyncService) Start() error {
	_, err := s.cron.AddFunc(s.cfg.SyncCron, func() {
		go s.Sync()
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	s.cron.Start()
	log.Printf("Sync service started with schedule: %s", s.cfg.SyncCron)

	// Run initial sync
	go s.Sync()

	return nil
}

func (s *SyncService) Stop() {
	s.cron.Stop()
}

// ForceSync triggers a sync immediately, even if one is already running
func (s *SyncService) ForceSync() error {
	s.mu.Lock()
	s.isRunning = false // Allow force sync even if one is running
	s.mu.Unlock()

	log.Println("Force sync triggered...")
	go s.Sync()
	return nil
}

// IsRunning returns whether a sync is currently in progress
func (s *SyncService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isRunning
}

func (s *SyncService) Sync() {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		log.Println("Sync already running, skipping...")
		return
	}
	s.isRunning = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
	}()

	log.Println("Starting data sync from GitHub ZIP archive...")

	ctx := context.Background()
	owner := "MatD1"
	repo := "arcraiders-data-fork"
	branch := "main"

	// 1. Get latest SHA to help with identification
	sha, err := s.getLatestSHA(ctx, owner, repo, branch)
	if err != nil {
		log.Printf("Warning: Could not get latest SHA: %v. Proceeding with sync anyway.", err)
	} else {
		log.Printf("Latest repository SHA: %s", sha)
	}

	// 2. Download zipball
	zipData, err := s.downloadArchive(ctx, owner, repo, branch)
	if err != nil {
		log.Printf("Error downloading archive: %v", err)
		return
	}
	log.Printf("Downloaded archive (%d bytes)", len(zipData))

	// 3. Process archive
	if err := s.processArchive(ctx, zipData); err != nil {
		log.Printf("Error processing archive: %v", err)
		return
	}

	log.Println("Data sync completed successfully.")

	// Update cache if available
	if s.dataCacheService != nil {
		log.Println("Triggering cache refresh...")
		s.dataCacheService.RefreshNow()
	}
}

func (s *SyncService) getLatestSHA(ctx context.Context, owner, repo, branch string) (string, error) {
	ref, _, err := s.githubClient.Git.GetRef(ctx, owner, repo, "heads/"+branch)
	if err != nil {
		return "", err
	}
	return ref.Object.GetSHA(), nil
}

func (s *SyncService) downloadArchive(ctx context.Context, owner, repo, ref string) ([]byte, error) {
	url, _, err := s.githubClient.Repositories.GetArchiveLink(ctx, owner, repo, github.Zipball, &github.RepositoryContentGetOptions{
		Ref: ref,
	}, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get archive link: %w", err)
	}

	resp, err := http.Get(url.String())
	if err != nil {
		return nil, fmt.Errorf("failed to download archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (s *SyncService) processArchive(ctx context.Context, zipData []byte) error {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	if err := s.syncQuestsFromZip(ctx, r); err != nil {
		log.Printf("Error syncing quests from zip: %v", err)
	}
	if err := s.syncItemsFromZip(ctx, r); err != nil {
		log.Printf("Error syncing items from zip: %v", err)
	}
	if err := s.syncSkillNodesFromZip(ctx, r); err != nil {
		log.Printf("Error syncing skill nodes from zip: %v", err)
	}
	if err := s.syncHideoutModulesFromZip(ctx, r); err != nil {
		log.Printf("Error syncing hideout modules from zip: %v", err)
	}
	if err := s.syncBotsFromZip(ctx, r); err != nil {
		log.Printf("Error syncing bots from zip: %v", err)
	}
	if err := s.syncMapsFromZip(ctx, r); err != nil {
		log.Printf("Error syncing maps from zip: %v", err)
	}
	if err := s.syncTradersFromZip(ctx, r); err != nil {
		log.Printf("Error syncing traders from zip: %v", err)
	}
	if err := s.syncProjectsFromZip(ctx, r); err != nil {
		log.Printf("Error syncing projects from zip: %v", err)
	}

	return nil
}

func (s *SyncService) getZipFile(r *zip.Reader, path string) ([]byte, error) {
	// GitHub zipballs have a root directory like "owner-repo-sha/"
	// We need to find the file regardless of the root directory name
	for _, f := range r.File {
		// Skip directories
		if f.FileInfo().IsDir() {
			continue
		}

		// Check if the file path ends with the target path (after the root dir)
		if strings.HasSuffix(f.Name, "/"+path) || f.Name == path {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("file not found in archive: %s", path)
}

func (s *SyncService) getZipDirFiles(r *zip.Reader, dir string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	dirToken := "/" + dir + "/"
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		if strings.Contains(f.Name, dirToken) && strings.HasSuffix(f.Name, ".json") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err == nil {
				// Use the filename as key
				parts := strings.Split(f.Name, "/")
				name := parts[len(parts)-1]
				files[name] = data
			}
		}
	}
	return files, nil
}

func (s *SyncService) loadZipCollection(r *zip.Reader, dir, fallback string) ([]map[string]interface{}, error) {
	// Try loading from directory first
	files, err := s.getZipDirFiles(r, dir)
	if err == nil && len(files) > 0 {
		var result []map[string]interface{}
		for _, data := range files {
			var decoded map[string]interface{}
			if err := json.Unmarshal(data, &decoded); err == nil {
				result = append(result, decoded)
			}
		}
		if len(result) > 0 {
			return result, nil
		}
	}

	// Fallback to single JSON file
	data, err := s.getZipFile(r, fallback)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetSnapshot returns a full point-in-time snapshot of all static data
func (s *SyncService) GetSnapshot() (*models.Snapshot, error) {
	snapshot := &models.Snapshot{
		Version:  "1.0", // Could be based on GitHub SHA or current timestamp
		SyncedAt: time.Now(),
	}

	var err error

	snapshot.Quests, err = s.questRepo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list quests: %w", err)
	}

	snapshot.Items, err = s.itemRepo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	snapshot.SkillNodes, err = s.skillNodeRepo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list skill nodes: %w", err)
	}

	snapshot.HideoutModules, err = s.hideoutModuleRepo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list hideout modules: %w", err)
	}

	snapshot.Bots, err = s.botRepo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list bots: %w", err)
	}

	snapshot.Maps, err = s.mapRepo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list maps: %w", err)
	}

	snapshot.Traders, err = s.traderRepo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list traders: %w", err)
	}

	snapshot.Projects, err = s.projectRepo.ListAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return snapshot, nil
}

func (s *SyncService) syncQuestsFromZip(ctx context.Context, r *zip.Reader) error {
	questsData, err := s.loadZipCollection(r, "quests", "quests.json")
	if err != nil {
		log.Printf("Warning: Could not load quests data from zip: %v", err)
		return nil
	}

	for _, q := range questsData {
		quest := &models.Quest{
			SyncedAt: time.Now(),
		}

		if id, ok := q["id"].(string); ok {
			quest.ExternalID = id
		} else if id, ok := q["id"].(float64); ok {
			quest.ExternalID = fmt.Sprintf("%.0f", id)
		}
		if name, ok := q["name"].(string); ok {
			quest.Name = name
		}
		if desc, ok := q["description"].(string); ok {
			quest.Description = desc
		}
		if trader, ok := q["trader"].(string); ok {
			quest.Trader = trader
		}
		if objectives, ok := q["objectives"].([]interface{}); ok {
			quest.Objectives = models.JSONB(map[string]interface{}{"objectives": objectives})
		}
		if rewardItemIds, ok := q["rewardItemIds"].([]interface{}); ok {
			quest.RewardItemIds = models.JSONB(map[string]interface{}{"reward_item_ids": rewardItemIds})
		}
		if xp, ok := q["xp"].(float64); ok {
			quest.XP = int(xp)
		}

		quest.Data = models.JSONB(q)

		err := s.questRepo.UpsertByExternalID(quest)
		if err != nil {
			log.Printf("Error upserting quest %s: %v", quest.ExternalID, err)
		}
	}

	log.Printf("Synced %d quests from zip", len(questsData))
	return nil
}

func (s *SyncService) syncItemsFromZip(ctx context.Context, r *zip.Reader) error {
	itemsData, err := s.loadZipCollection(r, "items", "items.json")
	if err != nil {
		log.Printf("Warning: Could not load items data from zip: %v", err)
		return nil
	}

	owner := "MatD1"
	repo := "arcraiders-data-fork"
	branch := "main"
	baseImageURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/images/items", owner, repo, branch)

	for _, i := range itemsData {
		item := &models.Item{
			SyncedAt: time.Now(),
		}

		if id, ok := i["id"].(string); ok {
			item.ExternalID = id
		} else if id, ok := i["id"].(float64); ok {
			item.ExternalID = fmt.Sprintf("%.0f", id)
		}
		if name, ok := i["name"].(string); ok {
			item.Name = name
		}
		if desc, ok := i["description"].(string); ok {
			item.Description = desc
		}
		if itemType, ok := i["type"].(string); ok {
			item.Type = itemType
		}

		var imagePath string
		if imgFilename, ok := i["imageFilename"].(string); ok && imgFilename != "" {
			item.ImageFilename = imgFilename
			imagePath = imgFilename
		} else if imgURL, ok := i["image_url"].(string); ok && imgURL != "" {
			imagePath = imgURL
		}

		if imagePath != "" {
			if strings.HasPrefix(imagePath, "http://") || strings.HasPrefix(imagePath, "https://") {
				item.ImageURL = imagePath
			} else {
				filename := strings.TrimPrefix(imagePath, "/")
				encodedFilename := url.PathEscape(filename)
				item.ImageURL = fmt.Sprintf("%s/%s", baseImageURL, encodedFilename)
			}
		}

		item.Data = models.JSONB(i)

		err := s.itemRepo.UpsertByExternalID(item)
		if err != nil {
			log.Printf("Error upserting item %s: %v", item.ExternalID, err)
		}
	}

	log.Printf("Synced %d items from zip", len(itemsData))
	return nil
}

func (s *SyncService) syncSkillNodesFromZip(ctx context.Context, r *zip.Reader) error {
	data, err := s.getZipFile(r, "skillNodes.json")
	if err != nil {
		log.Printf("Warning: Could not fetch skillNodes.json from zip: %v", err)
		return nil
	}

	var skillNodes []map[string]interface{}
	if err := json.Unmarshal(data, &skillNodes); err != nil {
		return err
	}

	for _, sn := range skillNodes {
		skillNode := &models.SkillNode{
			SyncedAt: time.Now(),
		}

		if id, ok := sn["id"].(string); ok {
			skillNode.ExternalID = id
		} else if id, ok := sn["id"].(float64); ok {
			skillNode.ExternalID = fmt.Sprintf("%.0f", id)
		}
		if name, ok := sn["name"].(string); ok {
			skillNode.Name = name
		}
		if desc, ok := sn["description"].(string); ok {
			skillNode.Description = desc
		}
		if impactedSkill, ok := sn["impactedSkill"].(string); ok {
			skillNode.ImpactedSkill = impactedSkill
		}
		if knownValue, ok := sn["knownValue"].([]interface{}); ok {
			skillNode.KnownValue = models.JSONB(map[string]interface{}{"known_value": knownValue})
		}
		if category, ok := sn["category"].(string); ok {
			skillNode.Category = category
		}
		if maxPoints, ok := sn["maxPoints"].(float64); ok {
			skillNode.MaxPoints = int(maxPoints)
		}
		if iconName, ok := sn["iconName"].(string); ok {
			skillNode.IconName = iconName
		}
		if isMajor, ok := sn["isMajor"].(bool); ok {
			skillNode.IsMajor = isMajor
		}
		if position, ok := sn["position"].(map[string]interface{}); ok {
			skillNode.Position = models.JSONB(position)
		}
		if prerequisiteNodeIds, ok := sn["prerequisiteNodeIds"].([]interface{}); ok {
			skillNode.PrerequisiteNodeIds = models.JSONB(map[string]interface{}{"prerequisite_node_ids": prerequisiteNodeIds})
		}

		skillNode.Data = models.JSONB(sn)

		err := s.skillNodeRepo.UpsertByExternalID(skillNode)
		if err != nil {
			log.Printf("Error upserting skill node %s: %v", skillNode.ExternalID, err)
		}
	}

	log.Printf("Synced %d skill nodes from zip", len(skillNodes))
	return nil
}

func (s *SyncService) syncHideoutModulesFromZip(ctx context.Context, r *zip.Reader) error {
	hideoutData, err := s.loadZipCollection(r, "hideout", "hideoutModules.json")
	if err != nil {
		log.Printf("Warning: Could not load hideout modules data from zip: %v", err)
		return nil
	}

	for _, hm := range hideoutData {
		hideoutModule := &models.HideoutModule{
			SyncedAt: time.Now(),
		}

		if id, ok := hm["id"].(string); ok {
			hideoutModule.ExternalID = id
		} else if id, ok := hm["id"].(float64); ok {
			hideoutModule.ExternalID = fmt.Sprintf("%.0f", id)
		}
		if name, ok := hm["name"].(string); ok {
			hideoutModule.Name = name
		}
		if desc, ok := hm["description"].(string); ok {
			hideoutModule.Description = desc
		}
		if maxLevel, ok := hm["maxLevel"].(float64); ok {
			hideoutModule.MaxLevel = int(maxLevel)
		}
		if levels, ok := hm["levels"].([]interface{}); ok {
			hideoutModule.Levels = models.JSONB(map[string]interface{}{"levels": levels})
		}

		hideoutModule.Data = models.JSONB(hm)

		err := s.hideoutModuleRepo.UpsertByExternalID(hideoutModule)
		if err != nil {
			log.Printf("Error upserting hideout module %s: %v", hideoutModule.ExternalID, err)
		}
	}

	log.Printf("Synced %d hideout modules from zip", len(hideoutData))
	return nil
}

func (s *SyncService) syncBotsFromZip(ctx context.Context, r *zip.Reader) error {
	data, err := s.getZipFile(r, "bots.json")
	if err != nil {
		log.Printf("Warning: Could not fetch bots.json from zip: %v", err)
		return nil
	}

	var bots []map[string]interface{}
	if err := json.Unmarshal(data, &bots); err != nil {
		return err
	}

	for _, b := range bots {
		bot := &models.Bot{
			SyncedAt: time.Now(),
		}

		if id, ok := b["id"].(string); ok {
			bot.ExternalID = id
		} else if id, ok := b["id"].(float64); ok {
			bot.ExternalID = fmt.Sprintf("%.0f", id)
		}
		if name, ok := b["name"].(string); ok {
			bot.Name = name
		}

		bot.Data = models.JSONB(b)

		err := s.botRepo.UpsertByExternalID(bot)
		if err != nil {
			log.Printf("Error upserting bot %s: %v", bot.ExternalID, err)
		}
	}

	log.Printf("Synced %d bots from zip", len(bots))
	return nil
}

func (s *SyncService) syncMapsFromZip(ctx context.Context, r *zip.Reader) error {
	data, err := s.getZipFile(r, "maps.json")
	if err != nil {
		log.Printf("Warning: Could not fetch maps.json from zip: %v", err)
		return nil
	}

	var maps []map[string]interface{}
	if err := json.Unmarshal(data, &maps); err != nil {
		return err
	}

	for _, m := range maps {
		mapModel := &models.Map{
			SyncedAt: time.Now(),
		}

		if id, ok := m["id"].(string); ok {
			mapModel.ExternalID = id
		} else if id, ok := m["id"].(float64); ok {
			mapModel.ExternalID = fmt.Sprintf("%.0f", id)
		}

		if name, ok := m["name"].(string); ok {
			mapModel.Name = name
		} else if nameObj, ok := m["name"].(map[string]interface{}); ok {
			if enName, ok := nameObj["en"].(string); ok && enName != "" {
				mapModel.Name = enName
			} else {
				for _, val := range nameObj {
					if nameStr, ok := val.(string); ok && nameStr != "" {
						mapModel.Name = nameStr
						break
					}
				}
			}
		}

		if mapModel.Name == "" {
			mapModel.Name = mapModel.ExternalID
		}

		mapModel.Data = models.JSONB(m)

		err := s.mapRepo.UpsertByExternalID(mapModel)
		if err != nil {
			log.Printf("Error upserting map %s: %v", mapModel.ExternalID, err)
		}
	}

	log.Printf("Synced %d maps from zip", len(maps))
	return nil
}

func (s *SyncService) syncTradersFromZip(ctx context.Context, r *zip.Reader) error {
	data, err := s.getZipFile(r, "trades.json")
	if err != nil {
		log.Printf("Warning: Could not fetch trades.json from zip: %v", err)
		return nil
	}

	var trades []map[string]interface{}
	if err := json.Unmarshal(data, &trades); err != nil {
		return err
	}

	traderMap := make(map[string]*models.Trader)

	for _, t := range trades {
		traderName, ok := t["trader"].(string)
		if !ok || traderName == "" {
			continue
		}

		externalID := traderName

		if trader, exists := traderMap[externalID]; exists {
			if trader.Data == nil {
				trader.Data = models.JSONB{"trades": []interface{}{t}}
			} else {
				dataMap := map[string]interface{}(trader.Data)
				if tradesArr, ok := dataMap["trades"].([]interface{}); ok {
					tradesArr = append(tradesArr, t)
					dataMap["trades"] = tradesArr
					trader.Data = models.JSONB(dataMap)
				}
			}
		} else {
			trader := &models.Trader{
				ExternalID: externalID,
				Name:       traderName,
				SyncedAt:   time.Now(),
				Data:       models.JSONB{"trades": []interface{}{t}},
			}
			traderMap[externalID] = trader
		}
	}

	for _, trader := range traderMap {
		err := s.traderRepo.UpsertByExternalID(trader)
		if err != nil {
			log.Printf("Error upserting trader %s: %v", trader.ExternalID, err)
		}
	}

	log.Printf("Synced %d traders from zip", len(traderMap))
	return nil
}

func (s *SyncService) syncProjectsFromZip(ctx context.Context, r *zip.Reader) error {
	data, err := s.getZipFile(r, "projects.json")
	if err != nil {
		log.Printf("Warning: Could not fetch projects.json from zip: %v", err)
		return nil
	}

	var projects []map[string]interface{}
	if err := json.Unmarshal(data, &projects); err != nil {
		return err
	}

	for _, p := range projects {
		project := &models.Project{
			SyncedAt: time.Now(),
		}

		if id, ok := p["id"].(string); ok {
			project.ExternalID = id
		} else if id, ok := p["id"].(float64); ok {
			project.ExternalID = fmt.Sprintf("%.0f", id)
		}

		if name, ok := p["name"].(string); ok {
			project.Name = name
		} else if nameObj, ok := p["name"].(map[string]interface{}); ok {
			if enName, ok := nameObj["en"].(string); ok && enName != "" {
				project.Name = enName
			} else {
				for _, val := range nameObj {
					if nameStr, ok := val.(string); ok && nameStr != "" {
						project.Name = nameStr
						break
					}
				}
			}
		}

		if project.Name == "" {
			project.Name = project.ExternalID
		}

		project.Data = models.JSONB(p)

		err := s.projectRepo.UpsertByExternalID(project)
		if err != nil {
			log.Printf("Error upserting project %s: %v", project.ExternalID, err)
		}
	}

	log.Printf("Synced %d projects from zip", len(projects))
	return nil
}
