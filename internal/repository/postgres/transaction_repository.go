package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"banking-service/internal/models"
)

// TransactionRepo is a PostgreSQL implementation of the repository.TransactionRepository interface
type TransactionRepo struct {
	db *sql.DB
}

// NewTransactionRepository creates a new TransactionRepo
func NewTransactionRepository(db *sql.DB) *TransactionRepo {
	return &TransactionRepo{db: db}
}

// Create creates a new transaction in the database
func (r *TransactionRepo) Create(ctx context.Context, transaction *models.Transaction) (int, error) {
	query := `INSERT INTO transactions (transaction_type, source_account_id, destination_account_id, 
             amount, currency, description, status, card_id, transaction_date) 
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	
	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		transaction.TransactionType,
		transaction.SourceAccountID,
		transaction.DestinationAccountID,
		transaction.Amount,
		transaction.Currency,
		transaction.Description,
		transaction.Status,
		transaction.CardID,
		transaction.TransactionDate,
	).Scan(&id)
	
	if err != nil {
		return 0, fmt.Errorf("failed to create transaction: %w", err)
	}
	
	return id, nil
}

// GetByID gets a transaction by ID
func (r *TransactionRepo) GetByID(ctx context.Context, id int) (*models.Transaction, error) {
	query := `SELECT id, transaction_type, source_account_id, destination_account_id, 
             amount, currency, description, status, card_id, transaction_date, created_at
             FROM transactions WHERE id = $1`
	
	transaction := &models.Transaction{}
	var sourceAccountID, destinationAccountID, cardID sql.NullInt32
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&transaction.ID,
		&transaction.TransactionType,
		&sourceAccountID,
		&destinationAccountID,
		&transaction.Amount,
		&transaction.Currency,
		&transaction.Description,
		&transaction.Status,
		&cardID,
		&transaction.TransactionDate,
		&transaction.CreatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("transaction not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	
	// Convert nullable fields
	if sourceAccountID.Valid {
		sID := int(sourceAccountID.Int32)
		transaction.SourceAccountID = &sID
	}
	
	if destinationAccountID.Valid {
		dID := int(destinationAccountID.Int32)
		transaction.DestinationAccountID = &dID
	}
	
	if cardID.Valid {
		cID := int(cardID.Int32)
		transaction.CardID = &cID
	}
	
	return transaction, nil
}

// GetByAccountID gets all transactions for an account
func (r *TransactionRepo) GetByAccountID(ctx context.Context, accountID int) ([]*models.Transaction, error) {
	query := `SELECT id, transaction_type, source_account_id, destination_account_id, 
             amount, currency, description, status, card_id, transaction_date, created_at
             FROM transactions 
             WHERE source_account_id = $1 OR destination_account_id = $1
             ORDER BY transaction_date DESC`
	
	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	defer rows.Close()
	
	return r.scanTransactions(rows)
}

// GetByUserID gets all transactions for a user through their accounts
func (r *TransactionRepo) GetByUserID(ctx context.Context, userID int) ([]*models.Transaction, error) {
	query := `SELECT t.id, t.transaction_type, t.source_account_id, t.destination_account_id, 
             t.amount, t.currency, t.description, t.status, t.card_id, t.transaction_date, t.created_at
             FROM transactions t
             JOIN accounts a ON t.source_account_id = a.id OR t.destination_account_id = a.id
             WHERE a.user_id = $1
             ORDER BY t.transaction_date DESC`
	
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	defer rows.Close()
	
	return r.scanTransactions(rows)
}

// GetByDateRange gets all transactions for a user within a date range
func (r *TransactionRepo) GetByDateRange(ctx context.Context, userID int, startDate, endDate time.Time) ([]*models.Transaction, error) {
	query := `SELECT t.id, t.transaction_type, t.source_account_id, t.destination_account_id, 
             t.amount, t.currency, t.description, t.status, t.card_id, t.transaction_date, t.created_at
             FROM transactions t
             JOIN accounts a ON t.source_account_id = a.id OR t.destination_account_id = a.id
             WHERE a.user_id = $1 AND t.transaction_date BETWEEN $2 AND $3
             ORDER BY t.transaction_date DESC`
	
	rows, err := r.db.QueryContext(ctx, query, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	defer rows.Close()
	
	return r.scanTransactions(rows)
}

// Update updates a transaction
func (r *TransactionRepo) Update(ctx context.Context, transaction *models.Transaction) error {
	query := `UPDATE transactions 
             SET status = $1, description = $2 
             WHERE id = $3`
	
	result, err := r.db.ExecContext(
		ctx,
		query,
		transaction.Status,
		transaction.Description,
		transaction.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return fmt.Errorf("transaction not found")
	}
	
	return nil
}

// Helper function to scan multiple transactions
func (r *TransactionRepo) scanTransactions(rows *sql.Rows) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	
	for rows.Next() {
		transaction := &models.Transaction{}
		var sourceAccountID, destinationAccountID, cardID sql.NullInt32
		
		err := rows.Scan(
			&transaction.ID,
			&transaction.TransactionType,
			&sourceAccountID,
			&destinationAccountID,
			&transaction.Amount,
			&transaction.Currency,
			&transaction.Description,
			&transaction.Status,
			&cardID,
			&transaction.TransactionDate,
			&transaction.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		
		// Convert nullable fields
		if sourceAccountID.Valid {
			sID := int(sourceAccountID.Int32)
			transaction.SourceAccountID = &sID
		}
		
		if destinationAccountID.Valid {
			dID := int(destinationAccountID.Int32)
			transaction.DestinationAccountID = &dID
		}
		
		if cardID.Valid {
			cID := int(cardID.Int32)
			transaction.CardID = &cID
		}
		
		transactions = append(transactions, transaction)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	
	return transactions, nil
}

// CreateTx creates a new transaction in the database within an existing transaction
func (r *TransactionRepo) CreateTx(ctx context.Context, tx *sql.Tx, transaction *models.Transaction) (int, error) {
	query := `INSERT INTO transactions (transaction_type, source_account_id, destination_account_id, 
             amount, currency, description, status, card_id, transaction_date) 
             VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	
	var id int
	err := tx.QueryRowContext(
		ctx,
		query,
		transaction.TransactionType,
		transaction.SourceAccountID,
		transaction.DestinationAccountID,
		transaction.Amount,
		transaction.Currency,
		transaction.Description,
		transaction.Status,
		transaction.CardID,
		transaction.TransactionDate,
	).Scan(&id)
	
	if err != nil {
		return 0, fmt.Errorf("failed to create transaction: %w", err)
	}
	
	return id, nil
}