package services

import (
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
	return user, err
}

func (s *UserService) GetByID(id uint) (*models.User, error) {
	return s.userRepo.FindByID(id)
}

func (s *UserService) IsAdmin(user *models.User) bool {
	return user.Role == models.RoleAdmin
}
