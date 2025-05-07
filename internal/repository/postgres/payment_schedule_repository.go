package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"banking-service/internal/models"
)

// PaymentScheduleRepo is a PostgreSQL implementation of the repository.PaymentScheduleRepository interface
type PaymentScheduleRepo struct {
	db *sql.DB
}

// NewPaymentScheduleRepository creates a new PaymentScheduleRepo
func NewPaymentScheduleRepository(db *sql.DB) *PaymentScheduleRepo {
	return &PaymentScheduleRepo{db: db}
}

// Create creates a new payment schedule item in the database
func (r *PaymentScheduleRepo) Create(ctx context.Context, schedule *models.PaymentSchedule) (int, error) {
	query := `INSERT INTO payment_schedules (credit_id, payment_date, principal_amount, 
             interest_amount, total_amount, status, is_overdue, penalty_amount) 
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	
	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		schedule.CreditID,
		schedule.PaymentDate,
		schedule.PrincipalAmount,
		schedule.InterestAmount,
		schedule.TotalAmount,
		schedule.Status,
		schedule.IsOverdue,
		schedule.PenaltyAmount,
	).Scan(&id)
	
	if err != nil {
		return 0, fmt.Errorf("failed to create payment schedule: %w", err)
	}
	
	return id, nil
}

// CreateBatch creates multiple payment schedule items in a single transaction
func (r *PaymentScheduleRepo) CreateBatch(ctx context.Context, schedules []*models.PaymentSchedule) error {
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
	
	// Prepare the SQL statement for batch insert
	valueStrings := make([]string, 0, len(schedules))
	valueArgs := make([]interface{}, 0, len(schedules)*8)
	
	for i, schedule := range schedules {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i*8+1, i*8+2, i*8+3, i*8+4, i*8+5, i*8+6, i*8+7, i*8+8))
		
		valueArgs = append(valueArgs, 
			schedule.CreditID,
			schedule.PaymentDate,
			schedule.PrincipalAmount,
			schedule.InterestAmount,
			schedule.TotalAmount,
			schedule.Status,
			schedule.IsOverdue,
			schedule.PenaltyAmount,
		)
	}
	
	stmt := fmt.Sprintf(`INSERT INTO payment_schedules 
                       (credit_id, payment_date, principal_amount, interest_amount, 
                        total_amount, status, is_overdue, penalty_amount) 
                       VALUES %s`, strings.Join(valueStrings, ","))
	
	_, err = tx.ExecContext(ctx, stmt, valueArgs...)
	if err != nil {
		return fmt.Errorf("failed to insert payment schedules: %w", err)
	}
	
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// GetByID gets a payment schedule item by ID
func (r *PaymentScheduleRepo) GetByID(ctx context.Context, id int) (*models.PaymentSchedule, error) {
	query := `SELECT id, credit_id, payment_date, principal_amount, interest_amount, 
             total_amount, status, is_overdue, penalty_amount, created_at, updated_at 
             FROM payment_schedules WHERE id = $1`
	
	schedule := &models.PaymentSchedule{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&schedule.ID,
		&schedule.CreditID,
		&schedule.PaymentDate,
		&schedule.PrincipalAmount,
		&schedule.InterestAmount,
		&schedule.TotalAmount,
		&schedule.Status,
		&schedule.IsOverdue,
		&schedule.PenaltyAmount,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("payment schedule not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get payment schedule: %w", err)
	}
	
	return schedule, nil
}

// GetByCreditID gets all payment schedule items for a credit
func (r *PaymentScheduleRepo) GetByCreditID(ctx context.Context, creditID int) ([]*models.PaymentSchedule, error) {
	query := `SELECT id, credit_id, payment_date, principal_amount, interest_amount, 
             total_amount, status, is_overdue, penalty_amount, created_at, updated_at 
             FROM payment_schedules 
             WHERE credit_id = $1
             ORDER BY payment_date`
	
	rows, err := r.db.QueryContext(ctx, query, creditID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment schedules: %w", err)
	}
	defer rows.Close()
	
	return r.scanPaymentSchedules(rows)
}

// Update updates a payment schedule item
func (r *PaymentScheduleRepo) Update(ctx context.Context, schedule *models.PaymentSchedule) error {
	query := `UPDATE payment_schedules 
             SET status = $1, is_overdue = $2, penalty_amount = $3 
             WHERE id = $4`
	
	result, err := r.db.ExecContext(
		ctx,
		query,
		schedule.Status,
		schedule.IsOverdue,
		schedule.PenaltyAmount,
		schedule.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update payment schedule: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return fmt.Errorf("payment schedule not found")
	}
	
	return nil
}

// GetPendingPayments gets all pending payments that are due on or before a specific date
func (r *PaymentScheduleRepo) GetPendingPayments(ctx context.Context, date time.Time) ([]*models.PaymentSchedule, error) {
	query := `SELECT ps.id, ps.credit_id, ps.payment_date, ps.principal_amount, ps.interest_amount, 
             ps.total_amount, ps.status, ps.is_overdue, ps.penalty_amount, ps.created_at, ps.updated_at,
             c.account_id
             FROM payment_schedules ps
             JOIN credits c ON ps.credit_id = c.id
             WHERE ps.status = $1 AND ps.payment_date <= $2
             ORDER BY ps.payment_date`
	
	rows, err := r.db.QueryContext(ctx, query, models.PaymentStatusPending, date)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending payments: %w", err)
	}
	defer rows.Close()
	
	var schedules []*models.PaymentSchedule
	
	for rows.Next() {
		schedule := &models.PaymentSchedule{}
		var accountID int
		
		err := rows.Scan(
			&schedule.ID,
			&schedule.CreditID,
			&schedule.PaymentDate,
			&schedule.PrincipalAmount,
			&schedule.InterestAmount,
			&schedule.TotalAmount,
			&schedule.Status,
			&schedule.IsOverdue,
			&schedule.PenaltyAmount,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
			&accountID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment schedule: %w", err)
		}
		
		schedules = append(schedules, schedule)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	
	return schedules, nil
}

// GetOverduePayments gets all overdue payments
func (r *PaymentScheduleRepo) GetOverduePayments(ctx context.Context) ([]*models.PaymentSchedule, error) {
	query := `SELECT id, credit_id, payment_date, principal_amount, interest_amount, 
             total_amount, status, is_overdue, penalty_amount, created_at, updated_at 
             FROM payment_schedules 
             WHERE status = $1 AND is_overdue = true
             ORDER BY payment_date`
	
	rows, err := r.db.QueryContext(ctx, query, models.PaymentStatusOverdue)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue payments: %w", err)
	}
	defer rows.Close()
	
	return r.scanPaymentSchedules(rows)
}

// Helper function to scan multiple payment schedules
func (r *PaymentScheduleRepo) scanPaymentSchedules(rows *sql.Rows) ([]*models.PaymentSchedule, error) {
	var schedules []*models.PaymentSchedule
	
	for rows.Next() {
		schedule := &models.PaymentSchedule{}
		err := rows.Scan(
			&schedule.ID,
			&schedule.CreditID,
			&schedule.PaymentDate,
			&schedule.PrincipalAmount,
			&schedule.InterestAmount,
			&schedule.TotalAmount,
			&schedule.Status,
			&schedule.IsOverdue,
			&schedule.PenaltyAmount,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment schedule: %w", err)
		}
		
		schedules = append(schedules, schedule)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	
	return schedules, nil
}