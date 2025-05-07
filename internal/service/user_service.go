package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"

	"banking-service/configs"
	"banking-service/internal/models"
	"banking-service/internal/repository"
	"banking-service/pkg/crypto"
)

// UserService is an implementation of the service.UserService interface
type UserSvc struct {
	repos      *repository.Repository
	logger     *logrus.Logger
	config     *configs.Config
	hasher     *crypto.PasswordHasher
	jwtSecret  string
	jwtTTL     time.Duration
}

// NewUserService creates a new UserSvc
func NewUserService(deps Dependencies) *UserSvc {
	return &UserSvc{
		repos:     deps.Repos,
		logger:    deps.Logger,
		config:    deps.Config,
		hasher:    crypto.NewPasswordHasher(),
		jwtSecret: deps.Config.JWT.Secret,
		jwtTTL:    time.Duration(deps.Config.JWT.TTL) * time.Hour,
	}
}

// Register registers a new user
func (s *UserSvc) Register(ctx context.Context, userReg *models.UserRegistration) (int, error) {
	// Validate user registration data
	if err := userReg.ValidateRegistration(); err != nil {
		return 0, fmt.Errorf("invalid user data: %w", err)
	}
	
	// Check if username already exists
	_, err := s.repos.User.GetByUsername(ctx, userReg.Username)
	if err == nil {
		return 0, errors.New("username already exists")
	}
	
	// Check if email already exists
	_, err = s.repos.User.GetByEmail(ctx, userReg.Email)
	if err == nil {
		return 0, errors.New("email already exists")
	}
	
	// Create a user object from registration data
	user := userReg.ToUser()
	
	// Hash the password
	hashedPassword, err := s.hasher.HashPassword(user.Password)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}
	
	user.PassHash = hashedPassword
	
	// Create the user in the database
	id, err := s.repos.User.Create(ctx, user)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}
	
	s.logger.Infof("User registered: %d", id)
	
	return id, nil
}

// Login logs in a user and returns a JWT token
func (s *UserSvc) Login(ctx context.Context, login *models.UserLogin) (*models.TokenResponse, error) {
	// Get user by username
	user, err := s.repos.User.GetByUsername(ctx, login.Username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	
	// Verify password
	if !s.hasher.CheckPasswordHash(login.Password, user.PassHash) {
		return nil, errors.New("invalid credentials")
	}
	
	// Generate JWT token
	expirationTime := time.Now().Add(s.jwtTTL)
	
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"exp":     expirationTime.Unix(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	// Sign the token with our secret
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	
	s.logger.Infof("User logged in: %d", user.ID)
	
	return &models.TokenResponse{
		Token:     tokenString,
		ExpiresAt: expirationTime.Unix(),
	}, nil
}

// GetByID gets a user by ID
func (s *UserSvc) GetByID(ctx context.Context, id int) (*models.User, error) {
	user, err := s.repos.User.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	// Don't expose the password hash
	user.PassHash = ""
	
	return user, nil
}

// Update updates a user
func (s *UserSvc) Update(ctx context.Context, user *models.User) error {
	// Get the original user to ensure the user exists
	originalUser, err := s.repos.User.GetByID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	// Check if username changed and if it's already taken
	if user.Username != originalUser.Username {
		existingUser, err := s.repos.User.GetByUsername(ctx, user.Username)
		if err == nil && existingUser.ID != user.ID {
			return errors.New("username already exists")
		}
	}
	
	// Check if email changed and if it's already taken
	if user.Email != originalUser.Email {
		existingUser, err := s.repos.User.GetByEmail(ctx, user.Email)
		if err == nil && existingUser.ID != user.ID {
			return errors.New("email already exists")
		}
	}
	
	// Update the user
	err = s.repos.User.Update(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	
	s.logger.Infof("User updated: %d", user.ID)
	
	return nil
}