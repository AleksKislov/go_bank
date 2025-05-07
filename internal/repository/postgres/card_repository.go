package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"banking-service/internal/models"
)

// CardRepo is a PostgreSQL implementation of the repository.CardRepository interface
type CardRepo struct {
	db *sql.DB
}

// NewCardRepository creates a new CardRepo
func NewCardRepository(db *sql.DB) *CardRepo {
	return &CardRepo{db: db}
}

// Create creates a new card in the database
func (r *CardRepo) Create(ctx context.Context, card *models.Card) (int, error) {
	query := `INSERT INTO cards (account_id, card_number_encrypted, card_number_hmac, 
             expiry_date_encrypted, cvv_hash, card_type, is_active) 
             VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	
	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		card.AccountID,
		card.CardNumberEncrypted,
		card.CardNumberHMAC,
		card.ExpiryDateEncrypted,
		card.CVVHash,
		card.CardType,
		card.IsActive,
	).Scan(&id)
	
	if err != nil {
		return 0, fmt.Errorf("failed to create card: %w", err)
	}
	
	return id, nil
}

// GetByID gets a card by ID
func (r *CardRepo) GetByID(ctx context.Context, id int) (*models.Card, error) {
	query := `SELECT id, account_id, card_number_encrypted, card_number_hmac, 
              expiry_date_encrypted, cvv_hash, card_type, is_active, created_at, updated_at 
              FROM cards WHERE id = $1`
	
	card := &models.Card{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&card.ID,
		&card.AccountID,
		&card.CardNumberEncrypted,
		&card.CardNumberHMAC,
		&card.ExpiryDateEncrypted,
		&card.CVVHash,
		&card.CardType,
		&card.IsActive,
		&card.CreatedAt,
		&card.UpdatedAt,
	)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("card not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get card: %w", err)
	}
	
	return card, nil
}

// GetByAccountID gets all cards for an account
func (r *CardRepo) GetByAccountID(ctx context.Context, accountID int) ([]*models.Card, error) {
	query := `SELECT id, account_id, card_number_encrypted, card_number_hmac, 
              expiry_date_encrypted, cvv_hash, card_type, is_active, created_at, updated_at 
              FROM cards WHERE account_id = $1`
	
	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cards: %w", err)
	}
	defer rows.Close()
	
	var cards []*models.Card
	for rows.Next() {
		card := &models.Card{}
		err := rows.Scan(
			&card.ID,
			&card.AccountID,
			&card.CardNumberEncrypted,
			&card.CardNumberHMAC,
			&card.ExpiryDateEncrypted,
			&card.CVVHash,
			&card.CardType,
			&card.IsActive,
			&card.CreatedAt,
			&card.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan card: %w", err)
		}
		cards = append(cards, card)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	
	return cards, nil
}

// GetByUserID gets all cards for a user through their accounts
func (r *CardRepo) GetByUserID(ctx context.Context, userID int) ([]*models.Card, error) {
	query := `SELECT c.id, c.account_id, c.card_number_encrypted, c.card_number_hmac, 
              c.expiry_date_encrypted, c.cvv_hash, c.card_type, c.is_active, c.created_at, c.updated_at 
              FROM cards c
              JOIN accounts a ON c.account_id = a.id
              WHERE a.user_id = $1`
	
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cards: %w", err)
	}
	defer rows.Close()
	
	var cards []*models.Card
	for rows.Next() {
		card := &models.Card{}
		err := rows.Scan(
			&card.ID,
			&card.AccountID,
			&card.CardNumberEncrypted,
			&card.CardNumberHMAC,
			&card.ExpiryDateEncrypted,
			&card.CVVHash,
			&card.CardType,
			&card.IsActive,
			&card.CreatedAt,
			&card.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan card: %w", err)
		}
		cards = append(cards, card)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	
	return cards, nil
}

// Update updates a card
func (r *CardRepo) Update(ctx context.Context, card *models.Card) error {
	query := `UPDATE cards 
              SET card_type = $1, is_active = $2 
              WHERE id = $3`
	
	result, err := r.db.ExecContext(
		ctx,
		query,
		card.CardType,
		card.IsActive,
		card.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update card: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return fmt.Errorf("card not found")
	}
	
	return nil
}

// Delete deletes a card (soft delete by setting is_active to false)
func (r *CardRepo) Delete(ctx context.Context, id int) error {
	query := `UPDATE cards SET is_active = false WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete card: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return fmt.Errorf("card not found")
	}
	
	return nil
}