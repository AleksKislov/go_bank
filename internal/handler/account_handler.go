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

// AccountHandler handles account-related HTTP requests
type AccountHandler struct {
	accountService service.AccountService
	logger         *logrus.Logger
	config         *configs.Config
}

// NewAccountHandler creates a new AccountHandler
func NewAccountHandler(accountService service.AccountService, logger *logrus.Logger, config *configs.Config) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
		logger:         logger,
		config:         config,
	}
}

// Create handles account creation
func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Parse request body
	var accountCreate models.AccountCreate
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&accountCreate); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	defer r.Body.Close()
	
	// Set the user ID from the authenticated user
	accountCreate.UserID = userID
	
	// Create the account
	accountID, err := h.accountService.Create(r.Context(), &accountCreate)
	if err != nil {
		h.logger.Warnf("Failed to create account: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusCreated, "account created successfully", map[string]interface{}{
		"account_id": accountID,
	})
}

// GetAll handles retrieving all accounts for a user
func (h *AccountHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get all accounts for the user
	accounts, err := h.accountService.GetByUserID(r.Context(), userID)
	if err != nil {
		h.logger.Warnf("Failed to get accounts: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get accounts")
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "accounts retrieved successfully", accounts)
}

// GetByID handles retrieving a specific account by ID
func (h *AccountHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get account ID from URL parameters
	vars := mux.Vars(r)
	accountID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid account ID")
		return
	}
	
	// Get the account
	account, err := h.accountService.GetByID(r.Context(), accountID, userID)
	if err != nil {
		h.logger.Warnf("Failed to get account: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "account not found")
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "account retrieved successfully", account)
}

// UpdateBalance handles deposit and withdrawal operations
func (h *AccountHandler) UpdateBalance(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get account ID from URL parameters
	vars := mux.Vars(r)
	accountID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid account ID")
		return
	}
	
	// Parse request body
	var balanceUpdate models.AccountBalance
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&balanceUpdate); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	defer r.Body.Close()
	
	// Determine if this is a deposit or withdrawal based on the amount
	var transactionID int
	
	if balanceUpdate.Amount > 0 {
		// Handle deposit
		depositRequest := &models.DepositRequest{
			AccountID:   accountID,
			Amount:      balanceUpdate.Amount,
			Description: balanceUpdate.Description,
		}
		
		transactionID, err = h.accountService.Deposit(r.Context(), accountID, userID, depositRequest)
	} else {
		utils.RespondWithError(w, http.StatusBadRequest, "amount must be positive")
		return
	}
	
	if err != nil {
		h.logger.Warnf("Failed to update balance: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "balance updated successfully", map[string]interface{}{
		"transaction_id": transactionID,
	})
}

// Delete handles account deletion
func (h *AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get account ID from URL parameters
	vars := mux.Vars(r)
	accountID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid account ID")
		return
	}
	
	// Delete the account
	err = h.accountService.Delete(r.Context(), accountID, userID)
	if err != nil {
		h.logger.Warnf("Failed to delete account: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "account deleted successfully", nil)
}