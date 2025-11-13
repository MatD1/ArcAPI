package services

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
)

const (
	itemsCacheKey       = "data:items:all"
	questsCacheKey      = "data:quests:all"
	dataCacheTTL        = 15 * time.Minute
	dataRefreshInterval = 15 * time.Minute
)

type DataCacheService struct {
	cacheService      *CacheService
	itemRepo          *repository.ItemRepository
	questRepo         *repository.QuestRepository
	mu                sync.RWMutex
	lastItemsRefresh  time.Time
	lastQuestsRefresh time.Time
}

func NewDataCacheService(
	cacheService *CacheService,
	itemRepo *repository.ItemRepository,
	questRepo *repository.QuestRepository,
) *DataCacheService {
	return &DataCacheService{
		cacheService: cacheService,
		itemRepo:     itemRepo,
		questRepo:    questRepo,
	}
}

// Start starts the background refresh goroutines
func (s *DataCacheService) Start() {
	// Initial refresh with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC recovered in initial refreshItems: %v", r)
			}
		}()
		s.refreshItems()
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC recovered in initial refreshQuests: %v", r)
			}
		}()
		s.refreshQuests()
	}()

	// Set up periodic refresh with panic recovery
	ticker := time.NewTicker(dataRefreshInterval)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC recovered in cache refresh ticker: %v", r)
			}
		}()
		for range ticker.C {
			// Wrap each refresh in its own recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("PANIC recovered in periodic refreshItems: %v", r)
					}
				}()
				s.refreshItems()
			}()

			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("PANIC recovered in periodic refreshQuests: %v", r)
					}
				}()
				s.refreshQuests()
			}()
		}
	}()
}

// refreshItems fetches all items from database and caches them
func (s *DataCacheService) refreshItems() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Fetch all items (with a large limit to get all)
	items, _, err := s.itemRepo.FindAll(0, 100000)
	if err != nil {
		fmt.Printf("Failed to fetch items for cache: %v\n", err)
		return
	}

	// Cache the items
	if err := s.cacheService.SetJSON(itemsCacheKey, items, dataCacheTTL); err != nil {
		fmt.Printf("Failed to cache items: %v\n", err)
		return
	}

	s.lastItemsRefresh = time.Now()
	fmt.Printf("Successfully refreshed items cache at %s (%d items)\n", s.lastItemsRefresh.Format(time.RFC3339), len(items))
}

// refreshQuests fetches all quests from database and caches them
func (s *DataCacheService) refreshQuests() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Fetch all quests
	quests, _, err := s.questRepo.FindAll(0, 100000)
	if err != nil {
		fmt.Printf("Failed to fetch quests for cache: %v\n", err)
		return
	}

	// Cache the quests
	if err := s.cacheService.SetJSON(questsCacheKey, quests, dataCacheTTL); err != nil {
		fmt.Printf("Failed to cache quests: %v\n", err)
		return
	}

	s.lastQuestsRefresh = time.Now()
	fmt.Printf("Successfully refreshed quests cache at %s (%d quests)\n", s.lastQuestsRefresh.Format(time.RFC3339), len(quests))
}

// GetItems returns cached items or fetches from database
func (s *DataCacheService) GetItems(offset, limit int) ([]models.Item, int64, error) {
	// Try to get from cache first
	var cachedItems []models.Item
	err := s.cacheService.GetJSON(itemsCacheKey, &cachedItems)
	if err == nil && len(cachedItems) > 0 {
		// Calculate total count
		total := int64(len(cachedItems))

		// Apply pagination
		end := offset + limit
		if end > len(cachedItems) {
			end = len(cachedItems)
		}
		if offset > len(cachedItems) {
			return []models.Item{}, total, nil
		}

		return cachedItems[offset:end], total, nil
	}

	// Cache miss - fetch from database
	items, count, err := s.itemRepo.FindAll(offset, limit)
	if err != nil {
		return nil, 0, err
	}

	// Trigger background refresh if cache is stale
	if s.lastItemsRefresh.IsZero() || time.Since(s.lastItemsRefresh) > dataRefreshInterval {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC recovered in background refreshItems: %v", r)
				}
			}()
			s.refreshItems()
		}()
	}

	return items, count, nil
}

// GetQuests returns cached quests or fetches from database
func (s *DataCacheService) GetQuests() ([]models.Quest, int64, error) {
	// Try to get from cache first
	var cachedQuests []models.Quest
	err := s.cacheService.GetJSON(questsCacheKey, &cachedQuests)
	if err == nil && len(cachedQuests) > 0 {
		total := int64(len(cachedQuests))
		return cachedQuests, total, nil
	}

	// Cache miss - fetch from database
	quests, count, err := s.questRepo.FindAll(0, 1000000)
	if err != nil {
		return nil, 0, err
	}

	// Trigger background refresh if cache is stale
	if s.lastQuestsRefresh.IsZero() || time.Since(s.lastQuestsRefresh) > dataRefreshInterval {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC recovered in background refreshQuests: %v", r)
				}
			}()
			s.refreshQuests()
		}()
	}

	return quests, count, nil
}

// InvalidateItemsCache clears the items cache
func (s *DataCacheService) InvalidateItemsCache() error {
	return s.cacheService.Delete(itemsCacheKey)
}

// InvalidateQuestsCache clears the quests cache
func (s *DataCacheService) InvalidateQuestsCache() error {
	return s.cacheService.Delete(questsCacheKey)
}

// InvalidateAllCache clears both caches
func (s *DataCacheService) InvalidateAllCache() error {
	if err := s.InvalidateItemsCache(); err != nil {
		return err
	}
	return s.InvalidateQuestsCache()
}
