package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	cfg *config.Config,
) *SyncService {
	return NewSyncServiceWithCache(questRepo, itemRepo, skillNodeRepo, hideoutModuleRepo, nil, cfg)
}

func NewSyncServiceWithCache(
	questRepo *repository.QuestRepository,
	itemRepo *repository.ItemRepository,
	skillNodeRepo *repository.SkillNodeRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
	dataCacheService *DataCacheService,
	cfg *config.Config,
) *SyncService {
	// GitHub client without auth for public repo
	client := github.NewClient(nil)

	service := &SyncService{
		questRepo:         questRepo,
		itemRepo:          itemRepo,
		skillNodeRepo:     skillNodeRepo,
		hideoutModuleRepo: hideoutModuleRepo,
		dataCacheService:  dataCacheService,
		githubClient:      client,
		cfg:               cfg,
		cron:              cron.New(),
	}

	return service
}

// NewSyncServiceWithMissionRepo is deprecated, use NewSyncService instead
func NewSyncServiceWithMissionRepo(
	missionRepo *repository.MissionRepository,
	itemRepo *repository.ItemRepository,
	skillNodeRepo *repository.SkillNodeRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
	cfg *config.Config,
) *SyncService {
	return NewSyncService(missionRepo, itemRepo, skillNodeRepo, hideoutModuleRepo, cfg)
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

	log.Println("Starting data sync from GitHub...")

	ctx := context.Background()
	owner := "MatD1"
	repo := "arcraiders-data-fork"

	// Fetch files concurrently
	var wg sync.WaitGroup
	errorChan := make(chan error, 4)

	wg.Add(4)
	go func() {
		defer wg.Done()
		if err := s.syncQuests(ctx, owner, repo); err != nil {
			errorChan <- fmt.Errorf("quests sync error: %w", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := s.syncItems(ctx, owner, repo); err != nil {
			errorChan <- fmt.Errorf("items sync error: %w", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := s.syncSkillNodes(ctx, owner, repo); err != nil {
			errorChan <- fmt.Errorf("skill nodes sync error: %w", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := s.syncHideoutModules(ctx, owner, repo); err != nil {
			errorChan <- fmt.Errorf("hideout modules sync error: %w", err)
		}
	}()

	wg.Wait()
	close(errorChan)

	// Log any errors
	for err := range errorChan {
		log.Printf("Sync error: %v", err)
	}

	log.Println("Data sync completed")
}

func (s *SyncService) fetchJSONFile(ctx context.Context, owner, repo, path string) ([]byte, error) {
	fileContent, _, _, err := s.githubClient.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		return nil, err
	}

	if fileContent.GetType() != "file" {
		return nil, fmt.Errorf("path is not a file: %s", path)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return nil, err
	}

	// No need to decode base64 manually, content is already decoded as UTF-8.
	decoded := []byte(content)

	return decoded, nil
}

func (s *SyncService) syncQuests(ctx context.Context, owner, repo string) error {
	// Fetch quests.json (in root of repo)
	paths := []string{"quests.json"}

	var data []byte
	var err error
	for _, path := range paths {
		data, err = s.fetchJSONFile(ctx, owner, repo, path)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Printf("Warning: Could not fetch quests.json: %v", err)
		return nil // Non-fatal
	}

	var quests []map[string]interface{}
	if err := json.Unmarshal(data, &quests); err != nil {
		return err
	}

	for _, q := range quests {
		quest := &models.Quest{
			SyncedAt: time.Now(),
		}

		// Extract common fields
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
			// Store as array directly, but wrap for consistency with TypeScript types
			quest.Objectives = models.JSONB(map[string]interface{}{"objectives": objectives})
		}
		if rewardItemIds, ok := q["rewardItemIds"].([]interface{}); ok {
			quest.RewardItemIds = models.JSONB(map[string]interface{}{"reward_item_ids": rewardItemIds})
		}
		if xp, ok := q["xp"].(float64); ok {
			quest.XP = int(xp)
		}

		// Store full data as JSONB
		quest.Data = models.JSONB(q)

		err := s.questRepo.UpsertByExternalID(quest)
		if err != nil {
			log.Printf("Error upserting quest %s: %v", quest.ExternalID, err)
		}
	}

	log.Printf("Synced %d quests from quests.json", len(quests))

	// Invalidate quests cache after sync
	if s.dataCacheService != nil {
		if err := s.dataCacheService.InvalidateQuestsCache(); err != nil {
			log.Printf("Failed to invalidate quests cache: %v", err)
		} else {
			log.Println("Quests cache invalidated after sync")
		}
	}

	return nil
}

// syncMissions is deprecated, use syncQuests instead
func (s *SyncService) syncMissions(ctx context.Context, owner, repo string) error {
	return s.syncQuests(ctx, owner, repo)
}

func (s *SyncService) syncItems(ctx context.Context, owner, repo string) error {
	paths := []string{"items.json"}

	var data []byte
	var err error
	for _, path := range paths {
		data, err = s.fetchJSONFile(ctx, owner, repo, path)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Printf("Warning: Could not fetch items.json: %v", err)
		return nil
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}

	// Get the default branch - try to get repo info, fallback to "main"
	branch := "main"
	repoInfo, _, err := s.githubClient.Repositories.Get(ctx, owner, repo)
	if err == nil && repoInfo.DefaultBranch != nil {
		branch = *repoInfo.DefaultBranch
	}

	// Base URL for GitHub raw content (free CDN via GitHub raw)
	baseImageURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/images/items", owner, repo, branch)

	for _, i := range items {
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

		// Handle image - check both imageFilename and image_url
		var imagePath string
		var imageSource string
		if imgFilename, ok := i["imageFilename"].(string); ok && imgFilename != "" {
			item.ImageFilename = imgFilename
			imagePath = imgFilename
			imageSource = "imageFilename"
		} else if imgURL, ok := i["image_url"].(string); ok && imgURL != "" {
			imagePath = imgURL
			imageSource = "image_url"
		}

		// Convert to GitHub raw URL if needed
		if imagePath != "" {
			// If it's already a full URL (http/https), use it as-is
			if strings.HasPrefix(imagePath, "http://") || strings.HasPrefix(imagePath, "https://") {
				item.ImageURL = imagePath
			} else {
				// Otherwise, treat it as a filename and construct GitHub raw URL
				filename := strings.TrimPrefix(imagePath, "/")
				// URL encode the filename in case it has special characters or spaces
				encodedFilename := url.PathEscape(filename)
				item.ImageURL = fmt.Sprintf("%s/%s", baseImageURL, encodedFilename)
			}
			log.Printf("Item %s: Image from %s -> %s", item.ExternalID, imageSource, item.ImageURL)
		} else {
			log.Printf("Item %s: No image found (checked imageFilename and image_url)", item.ExternalID)
		}

		item.Data = models.JSONB(i)

		err := s.itemRepo.UpsertByExternalID(item)
		if err != nil {
			log.Printf("Error upserting item %s: %v", item.ExternalID, err)
		}
	}

	log.Printf("Synced %d items", len(items))

	// Invalidate items cache after sync
	if s.dataCacheService != nil {
		if err := s.dataCacheService.InvalidateItemsCache(); err != nil {
			log.Printf("Failed to invalidate items cache: %v", err)
		} else {
			log.Println("Items cache invalidated after sync")
		}
	}

	return nil
}

func (s *SyncService) syncSkillNodes(ctx context.Context, owner, repo string) error {
	paths := []string{"skillNodes.json"}

	var data []byte
	var err error
	for _, path := range paths {
		data, err = s.fetchJSONFile(ctx, owner, repo, path)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Printf("Warning: Could not fetch skillNodes.json: %v", err)
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

	log.Printf("Synced %d skill nodes", len(skillNodes))
	return nil
}

func (s *SyncService) syncHideoutModules(ctx context.Context, owner, repo string) error {
	paths := []string{"hideoutModules.json"}

	var data []byte
	var err error
	for _, path := range paths {
		data, err = s.fetchJSONFile(ctx, owner, repo, path)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Printf("Warning: Could not fetch hideoutModules.json: %v", err)
		return nil
	}

	var hideoutModules []map[string]interface{}
	if err := json.Unmarshal(data, &hideoutModules); err != nil {
		return err
	}

	for _, hm := range hideoutModules {
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

	log.Printf("Synced %d hideout modules", len(hideoutModules))
	return nil
}
