package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"banking-service/configs"
	"banking-service/internal/models"
	"banking-service/internal/repository"
)

// TransactionSvc is an implementation of the service.TransactionService interface
type TransactionSvc struct {
	repos  *repository.Repository
	logger *logrus.Logger
	config *configs.Config
	email  EmailService
}

// NewTransactionService creates a new TransactionSvc
func NewTransactionService(deps Dependencies) *TransactionSvc {
	return &TransactionSvc{
		repos:  deps.Repos,
		logger: deps.Logger,
		config: deps.Config,
		email:  NewEmailService(deps),
	}
}

// Transfer performs a money transfer between accounts
func (s *TransactionSvc) Transfer(ctx context.Context, transfer *models.TransferRequest, userID int) (int, error) {
	// Validate transfer request
	if err := transfer.ValidateTransferRequest(); err != nil {
		return 0, fmt.Errorf("invalid transfer request: %w", err)
	}
	
	// Verify source account ownership
	sourceAccount, err := s.repos.Account.GetByID(ctx, transfer.SourceAccountID)
	if err != nil {
		return 0, fmt.Errorf("failed to get source account: %w", err)
	}
	
	if sourceAccount.UserID != userID {
		return 0, errors.New("access denied: source account belongs to another user")
	}
	
	// Check if source account is active
	if !sourceAccount.IsActive {
		return 0, errors.New("source account is inactive")
	}
	
	// Check if there are sufficient funds
	if sourceAccount.Balance < transfer.Amount {
		return 0, errors.New("insufficient funds")
	}
	
	// Get destination account (no ownership check required for destination)
	destAccount, err := s.repos.Account.GetByID(ctx, transfer.DestinationAccountID)
	if err != nil {
		return 0, fmt.Errorf("failed to get destination account: %w", err)
	}
	
	// Check if destination account is active
	if !destAccount.IsActive {
		return 0, errors.New("destination account is inactive")
	}
	
	// Check if currencies match
	if sourceAccount.Currency != destAccount.Currency {
		return 0, errors.New("currency mismatch between accounts")
	}
	
	// Start a transaction
	tx, err := s.repos.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	
	// Deduct from source account
	err = s.repos.Account.UpdateBalance(ctx, transfer.SourceAccountID, -transfer.Amount)
	if err != nil {
		return 0, fmt.Errorf("failed to update source account balance: %w", err)
	}
	
	// Add to destination account
	err = s.repos.Account.UpdateBalance(ctx, transfer.DestinationAccountID, transfer.Amount)
	if err != nil {
		return 0, fmt.Errorf("failed to update destination account balance: %w", err)
	}
	
	// Create transaction record
	transaction := transfer.ToTransaction()
	transaction.Currency = sourceAccount.Currency
	transaction.Status = models.TransactionStatusCompleted
	
	transactionID, err := s.repos.Transaction.Create(ctx, transaction)
	if err != nil {
		return 0, fmt.Errorf("failed to create transaction record: %w", err)
	}
	
	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	s.logger.Infof("Transfer of %f from account %d to account %d completed, transaction: %d", 
		transfer.Amount, transfer.SourceAccountID, transfer.DestinationAccountID, transactionID)
	
	// Send notification emails
	transaction.ID = transactionID
	go func() {
		ctx := context.Background()
		err := s.email.SendTransactionNotification(ctx, userID, transaction)
		if err != nil {
			s.logger.Warnf("Failed to send transaction notification: %v", err)
		}
	}()
	
	return transactionID, nil
}

// Pay processes a payment using a card
func (s *TransactionSvc) Pay(ctx context.Context, payment *models.PaymentRequest, userID int) (int, error) {
	// Validate payment request
	if err := payment.ValidatePaymentRequest(); err != nil {
		return 0, fmt.Errorf("invalid payment request: %w", err)
	}
	
	// Verify account ownership
	account, err := s.repos.Account.GetByID(ctx, payment.AccountID)
	if err != nil {
		return 0, fmt.Errorf("failed to get account: %w", err)
	}
	
	if account.UserID != userID {
		return 0, errors.New("access denied: account belongs to another user")
	}
	
	// Check if account is active
	if !account.IsActive {
		return 0, errors.New("account is inactive")
	}
	
	// Verify card ownership and status
	card, err := s.repos.Card.GetByID(ctx, payment.CardID)
	if err != nil {
		return 0, fmt.Errorf("failed to get card: %w", err)
	}
	
	if card.AccountID != payment.AccountID {
		return 0, errors.New("card does not belong to specified account")
	}
	
	if !card.IsActive {
		return 0, errors.New("card is inactive")
	}
	
	// Check if there are sufficient funds
	if account.Balance < payment.Amount {
		return 0, errors.New("insufficient funds")
	}
	
	// Start a transaction
	tx, err := s.repos.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	
	// Update account balance
	err = s.repos.Account.UpdateBalance(ctx, payment.AccountID, -payment.Amount)
	if err != nil {
		return 0, fmt.Errorf("failed to update account balance: %w", err)
	}
	
	// Create transaction record
	transaction := payment.ToTransaction()
	transaction.Currency = account.Currency
	transaction.Status = models.TransactionStatusCompleted
	
	transactionID, err := s.repos.Transaction.Create(ctx, transaction)
	if err != nil {
		return 0, fmt.Errorf("failed to create transaction record: %w", err)
	}
	
	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	s.logger.Infof("Payment of %f from account %d using card %d completed, transaction: %d", 
		payment.Amount, payment.AccountID, payment.CardID, transactionID)
	
	// Send notification email
	transaction.ID = transactionID
	go func() {
		ctx := context.Background()
		err := s.email.SendTransactionNotification(ctx, userID, transaction)
		if err != nil {
			s.logger.Warnf("Failed to send transaction notification: %v", err)
		}
	}()
	
	return transactionID, nil
}

// GetByID gets a transaction by ID and verifies ownership
func (s *TransactionSvc) GetByID(ctx context.Context, id int, userID int) (*models.Transaction, error) {
	// Get the transaction
	transaction, err := s.repos.Transaction.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	
	// Check ownership - either source or destination account must belong to user
	var accountIDs []int
	
	if transaction.SourceAccountID != nil {
		accountIDs = append(accountIDs, *transaction.SourceAccountID)
	}
	
	if transaction.DestinationAccountID != nil {
		accountIDs = append(accountIDs, *transaction.DestinationAccountID)
	}
	
	if len(accountIDs) == 0 {
		return nil, errors.New("invalid transaction: no source or destination account")
	}
	
	owned := false
	for _, accountID := range accountIDs {
		account, err := s.repos.Account.GetByID(ctx, accountID)
		if err != nil {
			continue
		}
		
		if account.UserID == userID {
			owned = true
			break
		}
	}
	
	if !owned {
		return nil, errors.New("access denied: transaction does not involve your accounts")
	}
	
	return transaction, nil
}

// GetByUserID gets all transactions for a user
func (s *TransactionSvc) GetByUserID(ctx context.Context, userID int) ([]*models.Transaction, error) {
	transactions, err := s.repos.Transaction.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	
	return transactions, nil
}

// GetByAccountID gets all transactions for an account and verifies ownership
func (s *TransactionSvc) GetByAccountID(ctx context.Context, accountID int, userID int) ([]*models.Transaction, error) {
	// Verify account ownership
	account, err := s.repos.Account.GetByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	
	if account.UserID != userID {
		return nil, errors.New("access denied: account belongs to another user")
	}
	
	// Get transactions for the account
	transactions, err := s.repos.Transaction.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	
	return transactions, nil
}

// GetByDateRange gets all transactions for a user within a date range
func (s *TransactionSvc) GetByDateRange(ctx context.Context, userID int, startDate, endDate time.Time) ([]*models.Transaction, error) {
	transactions, err := s.repos.Transaction.GetByDateRange(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	
	return transactions, nil
}