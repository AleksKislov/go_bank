package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"banking-service/internal/models"
)

// UserRepo is a PostgreSQL implementation of the repository.UserRepository interface
type UserRepo struct {
	db *sql.DB
}

// NewUserRepository creates a new UserRepo
func NewUserRepository(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create creates a new user in the database
func (r *UserRepo) Create(ctx context.Context, user *models.User) (int, error) {
	query := `INSERT INTO users (username, email, password_hash, first_name, last_name) 
			  VALUES ($1, $2, $3, $4, $5) RETURNING id`
	
	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Email,
		user.PassHash,
		user.FirstName,
		user.LastName,
	).Scan(&id)
	
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}
	
	return id, nil
}

// GetByID gets a user by ID
func (r *UserRepo) GetByID(ctx context.Context, id int) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, first_name, last_name, created_at, updated_at 
			  FROM users WHERE id = $1`
	
	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PassHash,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return user, nil
}

// GetByUsername gets a user by username
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, first_name, last_name, created_at, updated_at 
			  FROM users WHERE username = $1`
	
	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PassHash,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return user, nil
}

// GetByEmail gets a user by email
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, username, email, password_hash, first_name, last_name, created_at, updated_at 
			  FROM users WHERE email = $1`
	
	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PassHash,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return user, nil
}

// Update updates a user
func (r *UserRepo) Update(ctx context.Context, user *models.User) error {
	query := `UPDATE users 
			  SET username = $1, email = $2, first_name = $3, last_name = $4 
			  WHERE id = $5`
	
	result, err := r.db.ExecContext(
		ctx,
		query,
		user.Username,
		user.Email,
		user.FirstName,
		user.LastName,
		user.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	
	return nil
}

// Delete deletes a user by ID
func (r *UserRepo) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM users WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	
	return nil
}