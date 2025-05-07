package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"banking-service/internal/models"
)

// CreditRepo is a PostgreSQL implementation of the repository.CreditRepository interface
type CreditRepo struct {
	db *sql.DB
}

// NewCreditRepository creates a new CreditRepo
func NewCreditRepository(db *sql.DB) *CreditRepo {
	return &CreditRepo{db: db}
}

// Create creates a new credit in the database
func (r *CreditRepo) Create(ctx context.Context, credit *models.Credit) (int, error) {
	query := `INSERT INTO credits (user_id, account_id, amount, interest_rate, term_months, 
             monthly_payment, start_date, end_date, status) 
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	
	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		credit.UserID,
		credit.AccountID,
		credit.Amount,
		credit.InterestRate,
		credit.TermMonths,
		credit.MonthlyPayment,
		credit.StartDate,
		credit.EndDate,
		credit.Status,
	).Scan(&id)
	
	if err != nil {
		return 0, fmt.Errorf("failed to create credit: %w", err)
	}
	
	return id, nil
}

// GetByID gets a credit by ID
func (r *CreditRepo) GetByID(ctx context.Context, id int) (*models.Credit, error) {
	query := `SELECT id, user_id, account_id, amount, interest_rate, term_months, 
             monthly_payment, start_date, end_date, status, created_at, updated_at 
             FROM credits WHERE id = $1`
	
	credit := &models.Credit{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&credit.ID,
		&credit.UserID,
		&credit.AccountID,
		&credit.Amount,
		&credit.InterestRate,
		&credit.TermMonths,
		&credit.MonthlyPayment,
		&credit.StartDate,
		&credit.EndDate,
		&credit.Status,
		&credit.CreatedAt,
		&credit.UpdatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("credit not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get credit: %w", err)
	}
	
	return credit, nil
}

// GetByUserID gets all credits for a user
func (r *CreditRepo) GetByUserID(ctx context.Context, userID int) ([]*models.Credit, error) {
	query := `SELECT id, user_id, account_id, amount, interest_rate, term_months, 
             monthly_payment, start_date, end_date, status, created_at, updated_at 
             FROM credits WHERE user_id = $1
             ORDER BY created_at DESC`
	
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credits: %w", err)
	}
	defer rows.Close()
	
	return r.scanCredits(rows)
}

// GetByAccountID gets all credits for an account
func (r *CreditRepo) GetByAccountID(ctx context.Context, accountID int) ([]*models.Credit, error) {
	query := `SELECT id, user_id, account_id, amount, interest_rate, term_months, 
             monthly_payment, start_date, end_date, status, created_at, updated_at 
             FROM credits WHERE account_id = $1
             ORDER BY created_at DESC`
	
	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credits: %w", err)
	}
	defer rows.Close()
	
	return r.scanCredits(rows)
}

// Update updates a credit
func (r *CreditRepo) Update(ctx context.Context, credit *models.Credit) error {
	query := `UPDATE credits 
             SET status = $1, monthly_payment = $2
             WHERE id = $3`
	
	result, err := r.db.ExecContext(
		ctx,
		query,
		credit.Status,
		credit.MonthlyPayment,
		credit.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update credit: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return fmt.Errorf("credit not found")
	}
	
	return nil
}

// GetActiveCredits gets all active credits for automatic payment processing
func (r *CreditRepo) GetActiveCredits(ctx context.Context) ([]*models.Credit, error) {
	query := `SELECT id, user_id, account_id, amount, interest_rate, term_months, 
             monthly_payment, start_date, end_date, status, created_at, updated_at 
             FROM credits 
             WHERE status = $1
             ORDER BY created_at`
	
	rows, err := r.db.QueryContext(ctx, query, models.CreditStatusActive)
	if err != nil {
		return nil, fmt.Errorf("failed to get active credits: %w", err)
	}
	defer rows.Close()
	
	return r.scanCredits(rows)
}

// Helper function to scan multiple credits
func (r *CreditRepo) scanCredits(rows *sql.Rows) ([]*models.Credit, error) {
	var credits []*models.Credit
	
	for rows.Next() {
		credit := &models.Credit{}
		err := rows.Scan(
			&credit.ID,
			&credit.UserID,
			&credit.AccountID,
			&credit.Amount,
			&credit.InterestRate,
			&credit.TermMonths,
			&credit.MonthlyPayment,
			&credit.StartDate,
			&credit.EndDate,
			&credit.Status,
			&credit.CreatedAt,
			&credit.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan credit: %w", err)
		}
		
		credits = append(credits, credit)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	
	return credits, nil
}