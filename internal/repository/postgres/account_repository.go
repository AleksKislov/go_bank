package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"banking-service/internal/models"
)

// AccountRepo is a PostgreSQL implementation of the repository.AccountRepository interface
type AccountRepo struct {
	db *sql.DB
}

// NewAccountRepository creates a new AccountRepo
func NewAccountRepository(db *sql.DB) *AccountRepo {
	return &AccountRepo{db: db}
}

// Create creates a new account in the database
func (r *AccountRepo) Create(ctx context.Context, account *models.Account) (int, error) {
	query := `INSERT INTO accounts (user_id, account_number, balance, currency, account_type, is_active) 
			  VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	
	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		account.UserID,
		account.AccountNumber,
		account.Balance,
		account.Currency,
		account.AccountType,
		account.IsActive,
	).Scan(&id)
	
	if err != nil {
		return 0, fmt.Errorf("failed to create account: %w", err)
	}
	
	return id, nil
}

// GetByID gets an account by ID
func (r *AccountRepo) GetByID(ctx context.Context, id int) (*models.Account, error) {
	query := `SELECT id, user_id, account_number, balance, currency, account_type, is_active, created_at, updated_at 
			  FROM accounts WHERE id = $1`
	
	account := &models.Account{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&account.ID,
		&account.UserID,
		&account.AccountNumber,
		&account.Balance,
		&account.Currency,
		&account.AccountType,
		&account.IsActive,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("account not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	
	return account, nil
}

// GetByUserID gets all accounts for a user
func (r *AccountRepo) GetByUserID(ctx context.Context, userID int) ([]*models.Account, error) {
	query := `SELECT id, user_id, account_number, balance, currency, account_type, is_active, created_at, updated_at 
			  FROM accounts WHERE user_id = $1`
	
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}
	defer rows.Close()
	
	var accounts []*models.Account
	for rows.Next() {
		account := &models.Account{}
		err := rows.Scan(
			&account.ID,
			&account.UserID,
			&account.AccountNumber,
			&account.Balance,
			&account.Currency,
			&account.AccountType,
			&account.IsActive,
			&account.CreatedAt,
			&account.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, account)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	
	return accounts, nil
}

// GetByAccountNumber gets an account by account number
func (r *AccountRepo) GetByAccountNumber(ctx context.Context, accountNumber string) (*models.Account, error) {
	query := `SELECT id, user_id, account_number, balance, currency, account_type, is_active, created_at, updated_at 
			  FROM accounts WHERE account_number = $1`
	
	account := &models.Account{}
	err := r.db.QueryRowContext(ctx, query, accountNumber).Scan(
		&account.ID,
		&account.UserID,
		&account.AccountNumber,
		&account.Balance,
		&account.Currency,
		&account.AccountType,
		&account.IsActive,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("account not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	
	return account, nil
}

// UpdateBalance updates an account's balance
func (r *AccountRepo) UpdateBalance(ctx context.Context, id int, amount float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
	}()
	
	// First get the current balance to ensure it won't go negative
	query := `SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`
	var currentBalance float64
	
	err = tx.QueryRowContext(ctx, query, id).Scan(&currentBalance)
	if err != nil {
		return fmt.Errorf("failed to get current balance: %w", err)
	}
	
	newBalance := currentBalance + amount
	if newBalance < 0 {
		return fmt.Errorf("insufficient funds")
	}
	
	// Update the balance
	updateQuery := `UPDATE accounts SET balance = $1 WHERE id = $2`
	_, err = tx.ExecContext(ctx, updateQuery, newBalance, id)
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}
	
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// Update updates an account
func (r *AccountRepo) Update(ctx context.Context, account *models.Account) error {
	query := `UPDATE accounts 
			  SET currency = $1, account_type = $2, is_active = $3 
			  WHERE id = $4`
	
	result, err := r.db.ExecContext(
		ctx,
		query,
		account.Currency,
		account.AccountType,
		account.IsActive,
		account.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return fmt.Errorf("account not found")
	}
	
	return nil
}

// Delete deletes an account
func (r *AccountRepo) Delete(ctx context.Context, id int) error {
	// Start a transaction to ensure we don't delete accounts with a balance
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
	}()
	
	// Check if the account has a balance
	checkQuery := `SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`
	var balance float64
	
	err = tx.QueryRowContext(ctx, checkQuery, id).Scan(&balance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("account not found: %w", err)
		}
		return fmt.Errorf("failed to check account balance: %w", err)
	}
	
	if balance > 0 {
		return fmt.Errorf("cannot delete account with non-zero balance")
	}
	
	// Delete the account
	deleteQuery := `DELETE FROM accounts WHERE id = $1`
	result, err := tx.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return fmt.Errorf("account not found")
	}
	
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// UpdateBalanceTx updates an account's balance within an existing transaction
func (r *AccountRepo) UpdateBalanceTx(ctx context.Context, tx *sql.Tx, id int, amount float64) error {
	// First get the current balance to ensure it won't go negative
	query := `SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`
	var currentBalance float64
	
	err := tx.QueryRowContext(ctx, query, id).Scan(&currentBalance)
	if err != nil {
		return fmt.Errorf("failed to get current balance: %w", err)
	}
	
	newBalance := currentBalance + amount
	if newBalance < 0 {
		return fmt.Errorf("insufficient funds")
	}
	
	// Update the balance
	updateQuery := `UPDATE accounts SET balance = $1 WHERE id = $2`
	_, err = tx.ExecContext(ctx, updateQuery, newBalance, id)
	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}
	
	return nil
}