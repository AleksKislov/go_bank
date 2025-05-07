package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	"banking-service/configs"
	"banking-service/internal/models"
	"banking-service/internal/repository"
	"banking-service/pkg/crypto"
)

// CardSvc is an implementation of the service.CardService interface
type CardSvc struct {
	repos      *repository.Repository
	logger     *logrus.Logger
	config     *configs.Config
	pgp        *crypto.PGPCrypto
	hmac       *crypto.HMACSigner
	hasher     *crypto.PasswordHasher
}

// NewCardService creates a new CardSvc
func NewCardService(deps Dependencies) *CardSvc {
	pgpCrypto, err := crypto.NewPGPCrypto(
		deps.Config.PGP.PublicKey, 
		deps.Config.PGP.PrivateKey, 
		deps.Config.PGP.Passphrase,
	)
	if err != nil {
		deps.Logger.Warnf("Failed to initialize PGP crypto: %v. Using fallback.", err)
		pgpCrypto = crypto.NewFallbackPGPCrypto()
	}
	
	hmacSigner := crypto.NewHMACSigner([]byte(deps.Config.JWT.Secret))
	
	return &CardSvc{
		repos:      deps.Repos,
		logger:     deps.Logger,
		config:     deps.Config,
		pgp:        pgpCrypto,
		hmac:       hmacSigner,
		hasher:     crypto.NewPasswordHasher(),
	}
}

// Create creates a new card
func (s *CardSvc) Create(ctx context.Context, cardCreate *models.CardCreate, userID int) (int, error) {
	// Validate card creation data
	if err := cardCreate.ValidateCardCreate(); err != nil {
		return 0, fmt.Errorf("invalid card data: %w", err)
	}
	
	// Verify account ownership
	account, err := s.repos.Account.GetByID(ctx, cardCreate.AccountID)
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
	
	// Convert CardCreate to Card and generate card details
	card := cardCreate.ToCard()
	
	// Encrypt card number
	encryptedCardNumber, err := s.pgp.Encrypt(card.CardNumber)
	if err != nil {
		return 0, fmt.Errorf("failed to encrypt card number: %w", err)
	}
	card.CardNumberEncrypted = encryptedCardNumber
	
	// Create HMAC of card number for validation/lookup
	cardNumberHMAC := s.hmac.Sign(card.CardNumber)
	card.CardNumberHMAC = cardNumberHMAC
	
	// Encrypt expiry date
	encryptedExpiryDate, err := s.pgp.Encrypt(card.ExpiryDate)
	if err != nil {
		return 0, fmt.Errorf("failed to encrypt expiry date: %w", err)
	}
	card.ExpiryDateEncrypted = encryptedExpiryDate
	
	// Hash CVV (we never need to decrypt this)
	cvvHash, err := s.hasher.HashPassword(card.CVV)
	if err != nil {
		return 0, fmt.Errorf("failed to hash CVV: %w", err)
	}
	card.CVVHash = cvvHash
	
	// Create the card in the database
	id, err := s.repos.Card.Create(ctx, card)
	if err != nil {
		return 0, fmt.Errorf("failed to create card: %w", err)
	}
	
	s.logger.Infof("Card created: %d for account: %d", id, cardCreate.AccountID)
	
	return id, nil
}

// GetByID gets a card by ID and verifies ownership
func (s *CardSvc) GetByID(ctx context.Context, id int, userID int) (*models.CardResponse, error) {
	// Get the card
	card, err := s.repos.Card.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %w", err)
	}
	
	// Get the account to verify ownership
	account, err := s.repos.Account.GetByID(ctx, card.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	
	if account.UserID != userID {
		return nil, errors.New("access denied: card belongs to another user")
	}
	
	// Decrypt card number
	cardNumber, err := s.pgp.Decrypt(card.CardNumberEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt card number: %w", err)
	}
	card.CardNumber = cardNumber
	
	// Decrypt expiry date
	expiryDate, err := s.pgp.Decrypt(card.ExpiryDateEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt expiry date: %w", err)
	}
	card.ExpiryDate = expiryDate
	
	// Convert to response (masking the card number)
	response := card.ToCardResponse()
	
	return response, nil
}

// GetByUserID gets all cards for a user
func (s *CardSvc) GetByUserID(ctx context.Context, userID int) ([]*models.CardResponse, error) {
	// Get all cards for the user
	cards, err := s.repos.Card.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cards: %w", err)
	}
	
	// Process each card
	var responses []*models.CardResponse
	for _, card := range cards {
		// Decrypt card number
		cardNumber, err := s.pgp.Decrypt(card.CardNumberEncrypted)
		if err != nil {
			s.logger.Warnf("Failed to decrypt card number for card %d: %v", card.ID, err)
			continue
		}
		card.CardNumber = cardNumber
		
		// Decrypt expiry date
		expiryDate, err := s.pgp.Decrypt(card.ExpiryDateEncrypted)
		if err != nil {
			s.logger.Warnf("Failed to decrypt expiry date for card %d: %v", card.ID, err)
			continue
		}
		card.ExpiryDate = expiryDate
		
		// Convert to response (masking the card number)
		response := card.ToCardResponse()
		responses = append(responses, response)
	}
	
	return responses, nil
}

// GetByAccountID gets all cards for an account and verifies ownership
func (s *CardSvc) GetByAccountID(ctx context.Context, accountID int, userID int) ([]*models.CardResponse, error) {
	// Verify account ownership
	account, err := s.repos.Account.GetByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	
	if account.UserID != userID {
		return nil, errors.New("access denied: account belongs to another user")
	}
	
	// Get all cards for the account
	cards, err := s.repos.Card.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cards: %w", err)
	}
	
	// Process each card
	var responses []*models.CardResponse
	for _, card := range cards {
		// Decrypt card number
		cardNumber, err := s.pgp.Decrypt(card.CardNumberEncrypted)
		if err != nil {
			s.logger.Warnf("Failed to decrypt card number for card %d: %v", card.ID, err)
			continue
		}
		card.CardNumber = cardNumber
		
		// Decrypt expiry date
		expiryDate, err := s.pgp.Decrypt(card.ExpiryDateEncrypted)
		if err != nil {
			s.logger.Warnf("Failed to decrypt expiry date for card %d: %v", card.ID, err)
			continue
		}
		card.ExpiryDate = expiryDate
		
		// Convert to response (masking the card number)
		response := card.ToCardResponse()
		responses = append(responses, response)
	}
	
	return responses, nil
}

// Update updates a card (only status can be updated)
func (s *CardSvc) Update(ctx context.Context, card *models.Card, userID int) error {
	// Get the original card
	originalCard, err := s.repos.Card.GetByID(ctx, card.ID)
	if err != nil {
		return fmt.Errorf("failed to get card: %w", err)
	}
	
	// Verify ownership
	account, err := s.repos.Account.GetByID(ctx, originalCard.AccountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	
	if account.UserID != userID {
		return errors.New("access denied: card belongs to another user")
	}
	
	// Only allow updating isActive status
	updateCard := &models.Card{
		ID:       card.ID,
		IsActive: card.IsActive,
		CardType: originalCard.CardType,
	}
	
	// Update the card
	err = s.repos.Card.Update(ctx, updateCard)
	if err != nil {
		return fmt.Errorf("failed to update card: %w", err)
	}
	
	s.logger.Infof("Card updated: %d, active status: %v", card.ID, card.IsActive)
	
	return nil
}

// Delete deletes a card (sets it to inactive)
func (s *CardSvc) Delete(ctx context.Context, id int, userID int) error {
	// Get the card
	card, err := s.repos.Card.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get card: %w", err)
	}
	
	// Verify ownership
	account, err := s.repos.Account.GetByID(ctx, card.AccountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	
	if account.UserID != userID {
		return errors.New("access denied: card belongs to another user")
	}
	
	// Delete the card (soft delete)
	err = s.repos.Card.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete card: %w", err)
	}
	
	s.logger.Infof("Card deleted (deactivated): %d", id)
	
	return nil
}