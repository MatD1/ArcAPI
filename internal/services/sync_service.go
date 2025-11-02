package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/mat/arcapi/internal/config"
	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
	"github.com/robfig/cron/v3"
)

type SyncService struct {
	missionRepo       *repository.MissionRepository
	itemRepo          *repository.ItemRepository
	skillNodeRepo     *repository.SkillNodeRepository
	hideoutModuleRepo *repository.HideoutModuleRepository
	githubClient      *github.Client
	cfg               *config.Config
	cron              *cron.Cron
	mu                sync.Mutex
	isRunning         bool
}

func NewSyncService(
	missionRepo *repository.MissionRepository,
	itemRepo *repository.ItemRepository,
	skillNodeRepo *repository.SkillNodeRepository,
	hideoutModuleRepo *repository.HideoutModuleRepository,
	cfg *config.Config,
) *SyncService {
	// GitHub client without auth for public repo
	client := github.NewClient(nil)

	service := &SyncService{
		missionRepo:       missionRepo,
		itemRepo:          itemRepo,
		skillNodeRepo:     skillNodeRepo,
		hideoutModuleRepo: hideoutModuleRepo,
		githubClient:      client,
		cfg:               cfg,
		cron:              cron.New(),
	}

	return service
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
		if err := s.syncMissions(ctx, owner, repo); err != nil {
			errorChan <- fmt.Errorf("missions sync error: %w", err)
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

func (s *SyncService) syncMissions(ctx context.Context, owner, repo string) error {
	// Fetch quests.json for missions (in root of repo)
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

	var missions []map[string]interface{}
	if err := json.Unmarshal(data, &missions); err != nil {
		return err
	}

	for _, m := range missions {
		mission := &models.Mission{
			SyncedAt: time.Now(),
		}

		// Extract common fields
		if id, ok := m["id"].(string); ok {
			mission.ExternalID = id
		} else if id, ok := m["id"].(float64); ok {
			mission.ExternalID = fmt.Sprintf("%.0f", id)
		}
		if name, ok := m["name"].(string); ok {
			mission.Name = name
		}
		if desc, ok := m["description"].(string); ok {
			mission.Description = desc
		}

		// Store full data as JSONB
		mission.Data = models.JSONB(m)

		err := s.missionRepo.UpsertByExternalID(mission)
		if err != nil {
			log.Printf("Error upserting mission %s: %v", mission.ExternalID, err)
		}
	}

	log.Printf("Synced %d missions from quests.json", len(missions))
	return nil
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
		if img, ok := i["image_url"].(string); ok {
			item.ImageURL = img
		}

		item.Data = models.JSONB(i)

		err := s.itemRepo.UpsertByExternalID(item)
		if err != nil {
			log.Printf("Error upserting item %s: %v", item.ExternalID, err)
		}
	}

	log.Printf("Synced %d items", len(items))
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

		hideoutModule.Data = models.JSONB(hm)

		err := s.hideoutModuleRepo.UpsertByExternalID(hideoutModule)
		if err != nil {
			log.Printf("Error upserting hideout module %s: %v", hideoutModule.ExternalID, err)
		}
	}

	log.Printf("Synced %d hideout modules", len(hideoutModules))
	return nil
}
