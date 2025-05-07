package handler

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"banking-service/configs"
	"banking-service/internal/service"
	"banking-service/pkg/utils"
)

// AnalyticsHandler handles analytics-related HTTP requests
type AnalyticsHandler struct {
	analyticsService service.AnalyticsService
	logger           *logrus.Logger
	config           *configs.Config
}

// NewAnalyticsHandler creates a new AnalyticsHandler
func NewAnalyticsHandler(analyticsService service.AnalyticsService, logger *logrus.Logger, config *configs.Config) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
		logger:           logger,
		config:           config,
	}
}

// GetStatistics handles retrieving financial statistics for a user
func (h *AnalyticsHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get period from query parameters (default is "month")
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "month"
	}
	
	// Valid periods: week, month, quarter, year
	validPeriods := map[string]bool{
		"week":    true,
		"month":   true,
		"quarter": true,
		"year":    true,
	}
	
	if !validPeriods[period] {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid period. Must be one of: week, month, quarter, year")
		return
	}
	
	// Get the statistics
	statistics, err := h.analyticsService.GetStatistics(r.Context(), userID, period)
	if err != nil {
		h.logger.Warnf("Failed to get statistics: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get statistics")
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "statistics retrieved successfully", statistics)
}

// PredictBalance handles predicting future account balance
func (h *AnalyticsHandler) PredictBalance(w http.ResponseWriter, r *http.Request) {
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
	
	// Get days from query parameters (default is 30)
	daysStr := r.URL.Query().Get("days")
	days := 30
	
	if daysStr != "" {
		days, err = strconv.Atoi(daysStr)
		if err != nil || days <= 0 {
			utils.RespondWithError(w, http.StatusBadRequest, "invalid days parameter")
			return
		}
	}
	
	// Get the balance prediction
	prediction, err := h.analyticsService.PredictBalance(r.Context(), accountID, userID, days)
	if err != nil {
		h.logger.Warnf("Failed to predict balance: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to predict balance")
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "balance prediction retrieved successfully", prediction)
}

// GetCreditAnalytics handles retrieving credit analytics for a user
func (h *AnalyticsHandler) GetCreditAnalytics(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get credit analytics
	analytics, err := h.analyticsService.GetCreditAnalytics(r.Context(), userID)
	if err != nil {
		h.logger.Warnf("Failed to get credit analytics: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get credit analytics")
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "credit analytics retrieved successfully", analytics)
}