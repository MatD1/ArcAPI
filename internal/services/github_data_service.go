package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/go-github/v57/github"
)

const (
	githubDataRepoOwner = "MatD1"
	githubDataRepoName  = "arcraiders-data-fork"
	githubDataTTL       = 15 * time.Minute
)

var (
	botsPath     = "bots.json"
	mapsPath     = "maps.json"
	tradesPath   = "trades.json"
	projectsPath = "projects.json"
)

type GitHubDataService struct {
	cacheService *CacheService
	githubClient *github.Client
	owner        string
	repo         string
}

func NewGitHubDataService(cacheService *CacheService) *GitHubDataService {
	return &GitHubDataService{
		cacheService: cacheService,
		githubClient: github.NewClient(nil),
		owner:        githubDataRepoOwner,
		repo:         githubDataRepoName,
	}
}

func (s *GitHubDataService) GetBots(ctx context.Context) (interface{}, error) {
	return s.getJSON(ctx, "data:repo:bots", botsPath)
}

func (s *GitHubDataService) GetMaps(ctx context.Context) (interface{}, error) {
	return s.getJSON(ctx, "data:repo:maps", mapsPath)
}

func (s *GitHubDataService) GetTraders(ctx context.Context) (interface{}, error) {
	return s.getJSON(ctx, "data:repo:traders", tradesPath)
}

func (s *GitHubDataService) GetProjects(ctx context.Context) (interface{}, error) {
	return s.getJSON(ctx, "data:repo:projects", projectsPath)
}

func (s *GitHubDataService) getJSON(ctx context.Context, cacheKey, path string) (interface{}, error) {
	if s.cacheService != nil {
		var cached interface{}
		if err := s.cacheService.GetJSON(cacheKey, &cached); err == nil && cached != nil {
			return cached, nil
		}
	}

	raw, err := s.fetchFile(ctx, path)
	if err != nil {
		return nil, err
	}

	var parsed interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}

	if s.cacheService != nil {
		_ = s.cacheService.SetJSON(cacheKey, parsed, githubDataTTL)
	}

	return parsed, nil
}

func (s *GitHubDataService) fetchFile(ctx context.Context, path string) ([]byte, error) {
	fileContent, _, _, err := s.githubClient.Repositories.GetContents(ctx, s.owner, s.repo, path, nil)
	if err != nil {
		return nil, err
	}

	if fileContent.GetType() != "file" {
		return nil, fmt.Errorf("path is not a file: %s", path)
	}

	data, err := fileContent.GetContent()
	if err != nil {
		return nil, err
	}

	return []byte(data), nil
}
