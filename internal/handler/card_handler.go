package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"banking-service/configs"
	"banking-service/internal/models"
	"banking-service/internal/service"
	"banking-service/pkg/utils"
)

// CardHandler handles card-related HTTP requests
type CardHandler struct {
	cardService service.CardService
	logger      *logrus.Logger
	config      *configs.Config
}

// NewCardHandler creates a new CardHandler
func NewCardHandler(cardService service.CardService, logger *logrus.Logger, config *configs.Config) *CardHandler {
	return &CardHandler{
		cardService: cardService,
		logger:      logger,
		config:      config,
	}
}

// Create handles card creation
func (h *CardHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Parse request body
	var cardCreate models.CardCreate
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&cardCreate); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	defer r.Body.Close()
	
	// Create the card
	cardID, err := h.cardService.Create(r.Context(), &cardCreate, userID)
	if err != nil {
		h.logger.Warnf("Failed to create card: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusCreated, "card created successfully", map[string]interface{}{
		"card_id": cardID,
	})
}

// GetAll handles retrieving all cards for a user
func (h *CardHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Check if account ID is provided as a query parameter
	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr != "" {
		accountID, err := strconv.Atoi(accountIDStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid account ID")
			return
		}
		
		// Get cards for the specific account
		cards, err := h.cardService.GetByAccountID(r.Context(), accountID, userID)
		if err != nil {
			h.logger.Warnf("Failed to get cards for account: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to get cards")
			return
		}
		
		utils.RespondWithSuccess(w, http.StatusOK, "cards retrieved successfully", cards)
		return
	}
	
	// Get all cards for the user
	cards, err := h.cardService.GetByUserID(r.Context(), userID)
	if err != nil {
		h.logger.Warnf("Failed to get cards: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get cards")
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "cards retrieved successfully", cards)
}

// GetByID handles retrieving a specific card by ID
func (h *CardHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get card ID from URL parameters
	vars := mux.Vars(r)
	cardID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid card ID")
		return
	}
	
	// Get the card
	card, err := h.cardService.GetByID(r.Context(), cardID, userID)
	if err != nil {
		h.logger.Warnf("Failed to get card: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "card not found")
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "card retrieved successfully", card)
}

// Update handles updating card status
func (h *CardHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get card ID from URL parameters
	vars := mux.Vars(r)
	cardID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid card ID")
		return
	}
	
	// Parse request body
	var cardUpdate struct {
		IsActive bool `json:"is_active"`
	}
	
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&cardUpdate); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	defer r.Body.Close()
	
	// Create card object for update
	card := &models.Card{
		ID:       cardID,
		IsActive: cardUpdate.IsActive,
	}
	
	// Update the card
	err = h.cardService.Update(r.Context(), card, userID)
	if err != nil {
		h.logger.Warnf("Failed to update card: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "card updated successfully", nil)
}

// Delete handles card deletion (deactivation)
func (h *CardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get card ID from URL parameters
	vars := mux.Vars(r)
	cardID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid card ID")
		return
	}
	
	// Delete the card
	err = h.cardService.Delete(r.Context(), cardID, userID)
	if err != nil {
		h.logger.Warnf("Failed to delete card: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "card deleted successfully", nil)
}