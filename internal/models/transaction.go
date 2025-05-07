package models

import (
	"errors"
	"time"
)

// TransactionType defines the type of transaction
type TransactionType string

const (
	TransactionTypeDeposit    TransactionType = "DEPOSIT"
	TransactionTypeWithdrawal TransactionType = "WITHDRAWAL"
	TransactionTypeTransfer   TransactionType = "TRANSFER"
	TransactionTypePayment    TransactionType = "PAYMENT"
	TransactionTypeFee        TransactionType = "FEE"
	TransactionTypeInterest   TransactionType = "INTEREST"
)

// TransactionStatus defines the status of transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "PENDING"
	TransactionStatusCompleted TransactionStatus = "COMPLETED"
	TransactionStatusFailed    TransactionStatus = "FAILED"
	TransactionStatusCancelled TransactionStatus = "CANCELLED"
)

// Transaction represents a financial transaction
type Transaction struct {
	ID                  int               `json:"id" db:"id"`
	TransactionType     TransactionType   `json:"transaction_type" db:"transaction_type"`
	SourceAccountID     *int              `json:"source_account_id,omitempty" db:"source_account_id"`
	DestinationAccountID *int             `json:"destination_account_id,omitempty" db:"destination_account_id"`
	Amount              float64           `json:"amount" db:"amount"`
	Currency            Currency          `json:"currency" db:"currency"`
	Description         string            `json:"description,omitempty" db:"description"`
	Status              TransactionStatus `json:"status" db:"status"`
	CardID              *int              `json:"card_id,omitempty" db:"card_id"`
	TransactionDate     time.Time         `json:"transaction_date" db:"transaction_date"`
	CreatedAt           time.Time         `json:"created_at" db:"created_at"`
}

// TransferRequest represents a money transfer request
type TransferRequest struct {
	SourceAccountID      int     `json:"source_account_id" binding:"required"`
	DestinationAccountID int     `json:"destination_account_id" binding:"required"`
	Amount               float64 `json:"amount" binding:"required"`
	Description          string  `json:"description,omitempty"`
}

// DepositRequest represents a deposit request
type DepositRequest struct {
	AccountID    int     `json:"account_id" binding:"required"`
	Amount       float64 `json:"amount" binding:"required"`
	Description  string  `json:"description,omitempty"`
}

// WithdrawalRequest represents a withdrawal request
type WithdrawalRequest struct {
	AccountID    int     `json:"account_id" binding:"required"`
	Amount       float64 `json:"amount" binding:"required"`
	Description  string  `json:"description,omitempty"`
}

// PaymentRequest represents a payment request
type PaymentRequest struct {
	AccountID    int     `json:"account_id" binding:"required"`
	CardID       int     `json:"card_id" binding:"required"`
	Amount       float64 `json:"amount" binding:"required"`
	Description  string  `json:"description,omitempty"`
}

// ValidateTransferRequest validates transfer request data
func (t *TransferRequest) ValidateTransferRequest() error {
	if t.SourceAccountID == t.DestinationAccountID {
		return errors.New("source and destination accounts cannot be the same")
	}
	
	if t.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	
	return nil
}

// ToTransaction converts TransferRequest to Transaction
func (t *TransferRequest) ToTransaction() *Transaction {
	return &Transaction{
		TransactionType:      TransactionTypeTransfer,
		SourceAccountID:      &t.SourceAccountID,
		DestinationAccountID: &t.DestinationAccountID,
		Amount:               t.Amount,
		Currency:             CurrencyRUB, // Default currency, can be changed based on account
		Description:          t.Description,
		Status:               TransactionStatusPending,
		TransactionDate:      time.Now(),
	}
}

// ValidateDepositRequest validates deposit request data
func (d *DepositRequest) ValidateDepositRequest() error {
	if d.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	
	return nil
}

// ToTransaction converts DepositRequest to Transaction
func (d *DepositRequest) ToTransaction() *Transaction {
	return &Transaction{
		TransactionType:      TransactionTypeDeposit,
		DestinationAccountID: &d.AccountID,
		Amount:               d.Amount,
		Currency:             CurrencyRUB, // Default currency, can be changed based on account
		Description:          d.Description,
		Status:               TransactionStatusPending,
		TransactionDate:      time.Now(),
	}
}

// ValidateWithdrawalRequest validates withdrawal request data
func (w *WithdrawalRequest) ValidateWithdrawalRequest() error {
	if w.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	
	return nil
}

// ToTransaction converts WithdrawalRequest to Transaction
func (w *WithdrawalRequest) ToTransaction() *Transaction {
	return &Transaction{
		TransactionType:     TransactionTypeWithdrawal,
		SourceAccountID:     &w.AccountID,
		Amount:              w.Amount,
		Currency:            CurrencyRUB, // Default currency, can be changed based on account
		Description:         w.Description,
		Status:              TransactionStatusPending,
		TransactionDate:     time.Now(),
	}
}

// ValidatePaymentRequest validates payment request data
func (p *PaymentRequest) ValidatePaymentRequest() error {
	if p.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	
	return nil
}

// ToTransaction converts PaymentRequest to Transaction
func (p *PaymentRequest) ToTransaction() *Transaction {
	return &Transaction{
		TransactionType:     TransactionTypePayment,
		SourceAccountID:     &p.AccountID,
		Amount:              p.Amount,
		Currency:            CurrencyRUB, // Default currency, can be changed based on account
		Description:         p.Description,
		Status:              TransactionStatusPending,
		CardID:              &p.CardID,
		TransactionDate:     time.Now(),
	}
}