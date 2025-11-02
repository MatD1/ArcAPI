package services

import (
	"log"
	"strings"

	"github.com/mat/arcapi/internal/models"
	"github.com/mat/arcapi/internal/repository"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

func (s *UserService) CreateOrUpdateFromGithub(githubID string, email, username string) (*models.User, error) {
	// Try to find by GitHub ID
	user, err := s.userRepo.FindByGithubID(githubID)
	if err == nil {
		// Update if found
		user.Email = email
		user.Username = username
		err = s.userRepo.Update(user)
		return user, err
	}

	// Try to find by email
	user, err = s.userRepo.FindByEmail(email)
	if err == nil {
		// Update GitHub ID
		user.GithubID = &githubID
		err = s.userRepo.Update(user)
		return user, err
	}

	// Create new user
	user = &models.User{
		GithubID: &githubID,
		Email:    email,
		Username: username,
		Role:     models.RoleUser,
	}
	err = s.userRepo.Create(user)
	if err != nil {
		// Check for unique constraint violations
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			// Try to find the user again (might have been created in another request)
			user, findErr := s.userRepo.FindByEmail(email)
			if findErr == nil {
				return user, nil
			}
			user, findErr = s.userRepo.FindByGithubID(githubID)
			if findErr == nil {
				return user, nil
			}
		}
		log.Printf("Error creating user: %v", err)
		return nil, err
	}

	// Ensure ID is set (GORM should set this, but verify)
	if user.ID == 0 {
		log.Printf("Warning: User created but ID is still 0")
		// Try to find the user we just created
		user, err = s.userRepo.FindByEmail(email)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *UserService) GetByID(id uint) (*models.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *UserService) IsAdmin(user *models.User) bool {
	return user.Role == models.RoleAdmin
}
