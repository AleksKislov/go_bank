package repository

import (
	"context"
	"database/sql"
	"time"

	"banking-service/internal/models"
	"banking-service/internal/repository/postgres"
)

// TransactionManager defines methods for transaction management
type TransactionManager interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CommitTx(tx *sql.Tx) error
	RollbackTx(tx *sql.Tx) error
}

// UserRepository defines methods for user repository
type UserRepository interface {
	Create(ctx context.Context, user *models.User) (int, error)
	GetByID(ctx context.Context, id int) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id int) error
}

// AccountRepository defines methods for account repository
type AccountRepository interface {
	Create(ctx context.Context, account *models.Account) (int, error)
	GetByID(ctx context.Context, id int) (*models.Account, error)
	GetByUserID(ctx context.Context, userID int) ([]*models.Account, error)
	GetByAccountNumber(ctx context.Context, accountNumber string) (*models.Account, error)
	UpdateBalance(ctx context.Context, id int, amount float64) error
	Update(ctx context.Context, account *models.Account) error
	Delete(ctx context.Context, id int) error
	
	// Transaction-specific methods
	UpdateBalanceTx(ctx context.Context, tx *sql.Tx, id int, amount float64) error
}

// CardRepository defines methods for card repository
type CardRepository interface {
	Create(ctx context.Context, card *models.Card) (int, error)
	GetByID(ctx context.Context, id int) (*models.Card, error)
	GetByAccountID(ctx context.Context, accountID int) ([]*models.Card, error)
	GetByUserID(ctx context.Context, userID int) ([]*models.Card, error)
	Update(ctx context.Context, card *models.Card) error
	Delete(ctx context.Context, id int) error
}

// TransactionRepository defines methods for transaction repository
type TransactionRepository interface {
	Create(ctx context.Context, transaction *models.Transaction) (int, error)
	GetByID(ctx context.Context, id int) (*models.Transaction, error)
	GetByAccountID(ctx context.Context, accountID int) ([]*models.Transaction, error)
	GetByUserID(ctx context.Context, userID int) ([]*models.Transaction, error)
	GetByDateRange(ctx context.Context, userID int, startDate, endDate time.Time) ([]*models.Transaction, error)
	Update(ctx context.Context, transaction *models.Transaction) error
	
	// Transaction-specific methods
	CreateTx(ctx context.Context, tx *sql.Tx, transaction *models.Transaction) (int, error)
}

// CreditRepository defines methods for credit repository
type CreditRepository interface {
	Create(ctx context.Context, credit *models.Credit) (int, error)
	GetByID(ctx context.Context, id int) (*models.Credit, error)
	GetByUserID(ctx context.Context, userID int) ([]*models.Credit, error)
	GetByAccountID(ctx context.Context, accountID int) ([]*models.Credit, error)
	Update(ctx context.Context, credit *models.Credit) error
	GetActiveCredits(ctx context.Context) ([]*models.Credit, error)
}

// PaymentScheduleRepository defines methods for payment schedule repository
type PaymentScheduleRepository interface {
	Create(ctx context.Context, schedule *models.PaymentSchedule) (int, error)
	CreateBatch(ctx context.Context, schedules []*models.PaymentSchedule) error
	GetByID(ctx context.Context, id int) (*models.PaymentSchedule, error)
	GetByCreditID(ctx context.Context, creditID int) ([]*models.PaymentSchedule, error)
	Update(ctx context.Context, schedule *models.PaymentSchedule) error
	GetPendingPayments(ctx context.Context, date time.Time) ([]*models.PaymentSchedule, error)
	GetOverduePayments(ctx context.Context) ([]*models.PaymentSchedule, error)
}

// Repository is a composition of all repositories
type Repository struct {
	DB             *sql.DB
	User           UserRepository
	Account        AccountRepository
	Card           CardRepository
	Transaction    TransactionRepository
	Credit         CreditRepository
	PaymentSchedule PaymentScheduleRepository
}

// NewRepository creates a new repository with all sub-repositories
func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		DB:             db,
		User:           postgres.NewUserRepository(db),
		Account:        postgres.NewAccountRepository(db),
		Card:           postgres.NewCardRepository(db),
		Transaction:    postgres.NewTransactionRepository(db),
		Credit:         postgres.NewCreditRepository(db),
		PaymentSchedule: postgres.NewPaymentScheduleRepository(db),
	}
}

// BeginTx begins a new transaction
func (r *Repository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.DB.BeginTx(ctx, nil)
}

// CommitTx commits a transaction
func (r *Repository) CommitTx(tx *sql.Tx) error {
	return tx.Commit()
}

// RollbackTx rolls back a transaction
func (r *Repository) RollbackTx(tx *sql.Tx) error {
	return tx.Rollback()
}