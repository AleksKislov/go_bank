package models

import (
	"errors"
	"math/rand"
	"time"
)

// AccountType defines the type of bank account
type AccountType string

const (
	AccountTypeChecking AccountType = "CHECKING"
	AccountTypeSavings  AccountType = "SAVINGS"
	AccountTypeCredit   AccountType = "CREDIT"
)

// Currency represents supported currencies
type Currency string

const (
	CurrencyRUB Currency = "RUB"
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
)

// Account represents a bank account
type Account struct {
	ID           int        `json:"id" db:"id"`
	UserID       int        `json:"user_id" db:"user_id"`
	AccountNumber string     `json:"account_number" db:"account_number"`
	Balance      float64    `json:"balance" db:"balance"`
	Currency     Currency   `json:"currency" db:"currency"`
	AccountType  AccountType `json:"account_type" db:"account_type"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// AccountCreate represents data for creating a new account
type AccountCreate struct {
	UserID      int        `json:"user_id" binding:"required"`
	Currency    Currency   `json:"currency" binding:"required"`
	AccountType AccountType `json:"account_type" binding:"required"`
	InitialBalance float64  `json:"initial_balance,omitempty"`
}

// AccountBalance represents a balance update request
type AccountBalance struct {
	Amount      float64 `json:"amount" binding:"required"`
	Description string  `json:"description,omitempty"`
}

// GenerateAccountNumber generates a random account number
func GenerateAccountNumber() string {
	rand.Seed(time.Now().UnixNano())
	
	// Format: 40817XXXXXXXXXX (16 digits)
	accountNumber := "40817"
	
	for i := 0; i < 12; i++ {
		accountNumber += string(rune('0' + rand.Intn(10)))
	}
	
	return accountNumber
}

// ValidateAccountCreate validates account creation data
func (a *AccountCreate) ValidateAccountCreate() error {
	// Validate AccountType
	switch a.AccountType {
	case AccountTypeChecking, AccountTypeSavings, AccountTypeCredit:
		// Valid account type
	default:
		return errors.New("invalid account type")
	}
	
	// Validate Currency
	switch a.Currency {
	case CurrencyRUB, CurrencyUSD, CurrencyEUR:
		// Valid currency
	default:
		return errors.New("invalid currency")
	}
	
	// Validate initial balance
	if a.InitialBalance < 0 {
		return errors.New("initial balance cannot be negative")
	}
	
	return nil
}

// ToAccount converts AccountCreate to Account
func (a *AccountCreate) ToAccount() *Account {
	return &Account{
		UserID:       a.UserID,
		AccountNumber: GenerateAccountNumber(),
		Balance:      a.InitialBalance,
		Currency:     a.Currency,
		AccountType:  a.AccountType,
		IsActive:     true,
	}
}

// ValidateBalanceUpdate validates a balance update request
func (a *AccountBalance) ValidateBalanceUpdate() error {
	if a.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	
	return nil
}