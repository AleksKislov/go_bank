package models

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// User represents a user in the system
type User struct {
	ID        int       `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"-"`
	PassHash  string    `json:"-" db:"password_hash"`
	FirstName string    `json:"first_name,omitempty" db:"first_name"`
	LastName  string    `json:"last_name,omitempty" db:"last_name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// UserRegistration represents user registration data
type UserRegistration struct {
	Username  string `json:"username" binding:"required"`
	Email     string `json:"email" binding:"required"`
	Password  string `json:"password" binding:"required"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// UserLogin represents user login data
type UserLogin struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse represents the JWT token response
type TokenResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// ValidateRegistration validates user registration data
func (u *UserRegistration) ValidateRegistration() error {
	// Validate username
	if len(u.Username) < 3 || len(u.Username) > 50 {
		return errors.New("username must be between 3 and 50 characters")
	}
	
	// Validate email
	emailPattern := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	matched, err := regexp.MatchString(emailPattern, u.Email)
	if err != nil || !matched {
		return errors.New("invalid email format")
	}
	
	// Validate password
	if len(u.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	
	hasUppercase := regexp.MustCompile(`[A-Z]`).MatchString(u.Password)
	hasLowercase := regexp.MustCompile(`[a-z]`).MatchString(u.Password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(u.Password)
	
	if !hasUppercase || !hasLowercase || !hasNumber {
		return errors.New("password must contain at least one uppercase letter, one lowercase letter, and one number")
	}
	
	// Sanitize inputs
	u.Username = strings.TrimSpace(u.Username)
	u.Email = strings.TrimSpace(u.Email)
	u.FirstName = strings.TrimSpace(u.FirstName)
	u.LastName = strings.TrimSpace(u.LastName)
	
	return nil
}

// ToUser converts UserRegistration to User
func (u *UserRegistration) ToUser() *User {
	return &User{
		Username:  u.Username,
		Email:     u.Email,
		Password:  u.Password,
		FirstName: u.FirstName,
		LastName:  u.LastName,
	}
}