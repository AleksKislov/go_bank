package models

import (
	"errors"
	"math/rand"
	"strings"
	"time"
)

// CardType defines the type of card
type CardType string

const (
	CardTypeVirtual CardType = "VIRTUAL"
	CardTypeDebit   CardType = "DEBIT"
	CardTypeCredit  CardType = "CREDIT"
)

// Card represents a bank card
type Card struct {
	ID                 int       `json:"id" db:"id"`
	AccountID          int       `json:"account_id" db:"account_id"`
	CardNumberEncrypted []byte    `json:"-" db:"card_number_encrypted"`
	CardNumberHMAC     string    `json:"-" db:"card_number_hmac"`
	CardNumber         string    `json:"card_number,omitempty" db:"-"`
	ExpiryDateEncrypted []byte    `json:"-" db:"expiry_date_encrypted"`
	ExpiryDate         string    `json:"expiry_date,omitempty" db:"-"`
	CVVHash            string    `json:"-" db:"cvv_hash"`
	CVV                string    `json:"cvv,omitempty" db:"-"`
	CardType           CardType  `json:"card_type" db:"card_type"`
	IsActive           bool      `json:"is_active" db:"is_active"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// CardCreate represents data for creating a new card
type CardCreate struct {
	AccountID int      `json:"account_id" binding:"required"`
	CardType  CardType `json:"card_type" binding:"required"`
}

// CardResponse represents a sanitized card response
type CardResponse struct {
	ID           int      `json:"id"`
	AccountID    int      `json:"account_id"`
	CardNumber   string   `json:"card_number"`
	ExpiryDate   string   `json:"expiry_date"`
	CardType     CardType `json:"card_type"`
	IsActive     bool     `json:"is_active"`
}

// GenerateCardNumber generates a valid card number (using Luhn algorithm)
func GenerateCardNumber() string {
	// MIR cards start with 2200-2204
	prefix := "2200"
	
	// Generate remaining 12 digits (total 16 digits)
	cardNumber := prefix
	for i := 0; i < 11; i++ {
		cardNumber += string(rune('0' + rand.Intn(10)))
	}
	
	// Apply Luhn algorithm to get the check digit
	sum := 0
	alternate := false
	
	// Process in reverse order
	for i := len(cardNumber) - 1; i >= 0; i-- {
		digit := int(cardNumber[i] - '0')
		
		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		
		sum += digit
		alternate = !alternate
	}
	
	// Calculate check digit (last digit)
	checkDigit := (10 - (sum % 10)) % 10
	cardNumber += string(rune('0' + checkDigit))
	
	return cardNumber
}

// GenerateExpiryDate generates a card expiry date (3 years from now)
func GenerateExpiryDate() string {
	now := time.Now()
	expiry := now.AddDate(3, 0, 0)
	return expiry.Format("01/06") // MM/YY format
}

// GenerateCVV generates a random 3-digit CVV
func GenerateCVV() string {
	cvv := ""
	for i := 0; i < 3; i++ {
		cvv += string(rune('0' + rand.Intn(10)))
	}
	return cvv
}

// ValidateCardCreate validates card creation data
func (c *CardCreate) ValidateCardCreate() error {
	// Validate CardType
	switch c.CardType {
	case CardTypeVirtual, CardTypeDebit, CardTypeCredit:
		// Valid card type
	default:
		return errors.New("invalid card type")
	}
	
	return nil
}

// ToCard converts CardCreate to Card
func (c *CardCreate) ToCard() *Card {
	return &Card{
		AccountID:   c.AccountID,
		CardNumber:  GenerateCardNumber(),
		ExpiryDate:  GenerateExpiryDate(),
		CVV:         GenerateCVV(),
		CardType:    c.CardType,
		IsActive:    true,
	}
}

// ToCardResponse converts Card to CardResponse with masked card number
func (c *Card) ToCardResponse() *CardResponse {
	maskedNumber := c.CardNumber
	if len(maskedNumber) >= 16 {
		maskedNumber = maskedNumber[:6] + strings.Repeat("*", 6) + maskedNumber[12:]
	}
	
	return &CardResponse{
		ID:           c.ID,
		AccountID:    c.AccountID,
		CardNumber:   maskedNumber,
		ExpiryDate:   c.ExpiryDate,
		CardType:     c.CardType,
		IsActive:     c.IsActive,
	}
}