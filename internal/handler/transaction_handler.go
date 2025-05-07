package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"banking-service/configs"
	"banking-service/internal/models"
	"banking-service/internal/service"
	"banking-service/pkg/utils"
)

// TransactionHandler handles transaction-related HTTP requests
type TransactionHandler struct {
	transactionService service.TransactionService
	logger             *logrus.Logger
	config             *configs.Config
}

// NewTransactionHandler creates a new TransactionHandler
func NewTransactionHandler(transactionService service.TransactionService, logger *logrus.Logger, config *configs.Config) *TransactionHandler {
	return &TransactionHandler{
		transactionService: transactionService,
		logger:             logger,
		config:             config,
	}
}

// Transfer handles money transfers between accounts
func (h *TransactionHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Parse request body
	var transferReq models.TransferRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&transferReq); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	defer r.Body.Close()
	
	// Execute the transfer
	transactionID, err := h.transactionService.Transfer(r.Context(), &transferReq, userID)
	if err != nil {
		h.logger.Warnf("Failed to execute transfer: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "transfer completed successfully", map[string]interface{}{
		"transaction_id": transactionID,
	})
}

// Pay handles card payments
func (h *TransactionHandler) Pay(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Parse request body
	var paymentReq models.PaymentRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&paymentReq); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	defer r.Body.Close()
	
	// Execute the payment
	transactionID, err := h.transactionService.Pay(r.Context(), &paymentReq, userID)
	if err != nil {
		h.logger.Warnf("Failed to execute payment: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "payment completed successfully", map[string]interface{}{
		"transaction_id": transactionID,
	})
}

// GetAll handles retrieving all transactions for a user
func (h *TransactionHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Check for query parameters for date range
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	
	// If date range is specified, get transactions by date range
	if startDateStr != "" && endDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid start date format")
			return
		}
		
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid end date format")
			return
		}
		
		// Add one day to end date to include transactions on that day
		endDate = endDate.AddDate(0, 0, 1)
		
		transactions, err := h.transactionService.GetByDateRange(r.Context(), userID, startDate, endDate)
		if err != nil {
			h.logger.Warnf("Failed to get transactions by date range: %v", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "failed to get transactions")
			return
		}
		
		utils.RespondWithSuccess(w, http.StatusOK, "transactions retrieved successfully", transactions)
		return
	}
	
	// Get all transactions for the user
	transactions, err := h.transactionService.GetByUserID(r.Context(), userID)
	if err != nil {
		h.logger.Warnf("Failed to get transactions: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get transactions")
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "transactions retrieved successfully", transactions)
}

// GetByID handles retrieving a specific transaction by ID
func (h *TransactionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get transaction ID from URL parameters
	vars := mux.Vars(r)
	transactionID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid transaction ID")
		return
	}
	
	// Get the transaction
	transaction, err := h.transactionService.GetByID(r.Context(), transactionID, userID)
	if err != nil {
		h.logger.Warnf("Failed to get transaction: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "transaction not found")
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "transaction retrieved successfully", transaction)
}

// GetByAccount handles retrieving all transactions for a specific account
func (h *TransactionHandler) GetByAccount(w http.ResponseWriter, r *http.Request) {
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
	
	// Get transactions for the account
	transactions, err := h.transactionService.GetByAccountID(r.Context(), accountID, userID)
	if err != nil {
		h.logger.Warnf("Failed to get transactions for account: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get transactions")
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "transactions retrieved successfully", transactions)
}