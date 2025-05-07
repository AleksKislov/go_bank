package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	"banking-service/configs"
	"banking-service/internal/models"
	"banking-service/internal/repository"
)

// AccountSvc is an implementation of the service.AccountService interface
type AccountSvc struct {
	repos  *repository.Repository
	logger *logrus.Logger
	config *configs.Config
}

// NewAccountService creates a new AccountSvc
func NewAccountService(deps Dependencies) *AccountSvc {
	return &AccountSvc{
		repos:  deps.Repos,
		logger: deps.Logger,
		config: deps.Config,
	}
}

// Create creates a new account
func (s *AccountSvc) Create(ctx context.Context, accountCreate *models.AccountCreate) (int, error) {
	// Validate account creation data
	if err := accountCreate.ValidateAccountCreate(); err != nil {
		return 0, fmt.Errorf("invalid account data: %w", err)
	}
	
	// Check if user exists
	_, err := s.repos.User.GetByID(ctx, accountCreate.UserID)
	if err != nil {
		return 0, fmt.Errorf("user not found: %w", err)
	}
	
	// Convert AccountCreate to Account
	account := accountCreate.ToAccount()
	
	// Create the account in the database
	id, err := s.repos.Account.Create(ctx, account)
	if err != nil {
		return 0, fmt.Errorf("failed to create account: %w", err)
	}
	
	s.logger.Infof("Account created: %d for user: %d", id, accountCreate.UserID)
	
	return id, nil
}

// GetByID gets an account by ID and verifies ownership
func (s *AccountSvc) GetByID(ctx context.Context, id int, userID int) (*models.Account, error) {
	account, err := s.repos.Account.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	
	// Verify ownership
	if account.UserID != userID {
		return nil, errors.New("access denied: account belongs to another user")
	}
	
	return account, nil
}

// GetByUserID gets all accounts for a user
func (s *AccountSvc) GetByUserID(ctx context.Context, userID int) ([]*models.Account, error) {
	accounts, err := s.repos.Account.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}
	
	return accounts, nil
}

// Deposit adds funds to an account
func (s *AccountSvc) Deposit(ctx context.Context, accountID int, userID int, deposit *models.DepositRequest) (int, error) {
	// Validate deposit request
	if err := deposit.ValidateDepositRequest(); err != nil {
		return 0, fmt.Errorf("invalid deposit request: %w", err)
	}
	
	// Verify account ownership
	account, err := s.GetByID(ctx, accountID, userID)
	if err != nil {
		return 0, err
	}
	
	// Check if account is active
	if !account.IsActive {
		return 0, errors.New("account is inactive")
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
	err = s.repos.Account.UpdateBalance(ctx, accountID, deposit.Amount)
	if err != nil {
		return 0, fmt.Errorf("failed to update balance: %w", err)
	}
	
	// Create transaction record
	transaction := deposit.ToTransaction()
	transactionID, err := s.repos.Transaction.Create(ctx, transaction)
	if err != nil {
		return 0, fmt.Errorf("failed to create transaction record: %w", err)
	}
	
	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	s.logger.Infof("Deposit of %f to account %d completed, transaction: %d", 
		deposit.Amount, accountID, transactionID)
	
	return transactionID, nil
}

// Withdraw removes funds from an account
func (s *AccountSvc) Withdraw(ctx context.Context, accountID int, userID int, withdrawal *models.WithdrawalRequest) (int, error) {
	// Validate withdrawal request
	if err := withdrawal.ValidateWithdrawalRequest(); err != nil {
		return 0, fmt.Errorf("invalid withdrawal request: %w", err)
	}
	
	// Verify account ownership
	account, err := s.GetByID(ctx, accountID, userID)
	if err != nil {
		return 0, err
	}
	
	// Check if account is active
	if !account.IsActive {
		return 0, errors.New("account is inactive")
	}
	
	// Check if there are sufficient funds
	if account.Balance < withdrawal.Amount {
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
	
	// Update account balance (negative amount for withdrawal)
	err = s.repos.Account.UpdateBalance(ctx, accountID, -withdrawal.Amount)
	if err != nil {
		return 0, fmt.Errorf("failed to update balance: %w", err)
	}
	
	// Create transaction record
	transaction := withdrawal.ToTransaction()
	transactionID, err := s.repos.Transaction.Create(ctx, transaction)
	if err != nil {
		return 0, fmt.Errorf("failed to create transaction record: %w", err)
	}
	
	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	s.logger.Infof("Withdrawal of %f from account %d completed, transaction: %d", 
		withdrawal.Amount, accountID, transactionID)
	
	return transactionID, nil
}

// Update updates an account
func (s *AccountSvc) Update(ctx context.Context, account *models.Account, userID int) error {
	// Verify account ownership
	originalAccount, err := s.GetByID(ctx, account.ID, userID)
	if err != nil {
		return err
	}
	
	// Prevent modification of critical fields
	account.UserID = originalAccount.UserID
	account.AccountNumber = originalAccount.AccountNumber
	account.Balance = originalAccount.Balance
	
	// Update the account
	err = s.repos.Account.Update(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}
	
	s.logger.Infof("Account updated: %d", account.ID)
	
	return nil
}

// Delete deletes an account
func (s *AccountSvc) Delete(ctx context.Context, id int, userID int) error {
	// Verify account ownership
	_, err := s.GetByID(ctx, id, userID)
	if err != nil {
		return err
	}
	
	// Check if account has active cards
	cards, err := s.repos.Card.GetByAccountID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get cards: %w", err)
	}
	
	for _, card := range cards {
		if card.IsActive {
			return errors.New("cannot delete account with active cards")
		}
	}
	
	// Delete the account
	err = s.repos.Account.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}
	
	s.logger.Infof("Account deleted: %d", id)
	
	return nil
}