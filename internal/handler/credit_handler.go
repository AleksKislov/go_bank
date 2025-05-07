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

// CreditHandler handles credit-related HTTP requests
type CreditHandler struct {
	creditService service.CreditService
	logger        *logrus.Logger
	config        *configs.Config
}

// NewCreditHandler creates a new CreditHandler
func NewCreditHandler(creditService service.CreditService, logger *logrus.Logger, config *configs.Config) *CreditHandler {
	return &CreditHandler{
		creditService: creditService,
		logger:        logger,
		config:        config,
	}
}

// Create handles credit creation
func (h *CreditHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Parse request body
	var creditRequest models.CreditRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&creditRequest); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	defer r.Body.Close()
	
	// Set the user ID from the authenticated user
	creditRequest.UserID = userID
	
	// Create the credit
	creditID, err := h.creditService.Create(r.Context(), &creditRequest)
	if err != nil {
		h.logger.Warnf("Failed to create credit: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusCreated, "credit created successfully", map[string]interface{}{
		"credit_id": creditID,
	})
}

// GetAll handles retrieving all credits for a user
func (h *CreditHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get all credits for the user
	credits, err := h.creditService.GetByUserID(r.Context(), userID)
	if err != nil {
		h.logger.Warnf("Failed to get credits: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get credits")
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "credits retrieved successfully", credits)
}

// GetByID handles retrieving a specific credit by ID
func (h *CreditHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get credit ID from URL parameters
	vars := mux.Vars(r)
	creditID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid credit ID")
		return
	}
	
	// Get the credit
	credit, err := h.creditService.GetByID(r.Context(), creditID, userID)
	if err != nil {
		h.logger.Warnf("Failed to get credit: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "credit not found")
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "credit retrieved successfully", credit)
}

// GetSchedule handles retrieving the payment schedule for a credit
func (h *CreditHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get credit ID from URL parameters
	vars := mux.Vars(r)
	creditID, err := strconv.Atoi(vars["id"])
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid credit ID")
		return
	}
	
	// Get the payment schedule
	schedule, summary, err := h.creditService.GetSchedule(r.Context(), creditID, userID)
	if err != nil {
		h.logger.Warnf("Failed to get payment schedule: %v", err)
		utils.RespondWithError(w, http.StatusNotFound, "payment schedule not found")
		return
	}
	
	// Return success response
	response := map[string]interface{}{
		"payments": schedule,
		"summary":  summary,
	}
	
	utils.RespondWithSuccess(w, http.StatusOK, "payment schedule retrieved successfully", response)
}

// GetKeyRate handles retrieving the current central bank key rate
func (h *CreditHandler) GetKeyRate(w http.ResponseWriter, r *http.Request) {
	// Get the key rate
	keyRate, err := h.creditService.GetKeyRate(r.Context())
	if err != nil {
		h.logger.Warnf("Failed to get key rate: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get key rate")
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "key rate retrieved successfully", map[string]interface{}{
		"key_rate": keyRate,
	})
}