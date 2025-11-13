package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	tradersCacheKey        = "traders:data"
	tradersAPIURL          = "https://metaforge.app/api/arc-raiders"
	tradersCacheTTL        = 15 * time.Minute
	tradersRefreshInterval = 15 * time.Minute
)

type TradersService struct {
	cacheService *CacheService
	httpClient   *http.Client
	mu           sync.RWMutex
	lastFetch    time.Time
}

func NewTradersService(cacheService *CacheService) *TradersService {
	return &TradersService{
		cacheService: cacheService,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start starts the background refresh goroutine
func (s *TradersService) Start() {
	// Initial fetch with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC recovered in initial refreshTraders: %v", r)
			}
		}()
		s.refreshTraders()
	}()

	// Set up periodic refresh with panic recovery
	ticker := time.NewTicker(tradersRefreshInterval)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC recovered in traders refresh ticker: %v", r)
			}
		}()
		for range ticker.C {
			// Wrap refresh in its own recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("PANIC recovered in periodic refreshTraders: %v", r)
					}
				}()
				s.refreshTraders()
			}()
		}
	}()
}

// refreshTraders fetches traders data from the external API and caches it
func (s *TradersService) refreshTraders() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Fetch from external API
	resp, err := s.httpClient.Get(tradersAPIURL)
	if err != nil {
		fmt.Printf("Failed to fetch traders data: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Failed to fetch traders data: status code %d\n", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read traders response: %v\n", err)
		return
	}

	// Validate JSON
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Printf("Failed to parse traders JSON: %v\n", err)
		return
	}

	// Cache the data
	if err := s.cacheService.SetJSON(tradersCacheKey, data, tradersCacheTTL); err != nil {
		fmt.Printf("Failed to cache traders data: %v\n", err)
		return
	}

	s.lastFetch = time.Now()
	fmt.Printf("Successfully refreshed traders data at %s\n", s.lastFetch.Format(time.RFC3339))
}

// GetTraders returns the cached traders data, fetching if necessary
func (s *TradersService) GetTraders() (interface{}, error) {
	// Try to get from cache first
	var cachedData interface{}
	err := s.cacheService.GetJSON(tradersCacheKey, &cachedData)
	if err == nil && cachedData != nil {
		return cachedData, nil
	}

	// Cache miss or error - fetch fresh data
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check cache after acquiring lock (another goroutine might have fetched)
	err = s.cacheService.GetJSON(tradersCacheKey, &cachedData)
	if err == nil && cachedData != nil {
		return cachedData, nil
	}

	// Still no cache - fetch now
	resp, err := s.httpClient.Get(tradersAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch traders data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch traders data: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read traders response: %w", err)
	}

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse traders JSON: %w", err)
	}

	// Cache for future requests
	s.cacheService.SetJSON(tradersCacheKey, data, tradersCacheTTL)
	s.lastFetch = time.Now()

	return data, nil
}
