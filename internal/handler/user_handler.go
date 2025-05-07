package handler

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"

	"banking-service/configs"
	"banking-service/internal/models"
	"banking-service/internal/service"
	"banking-service/pkg/utils"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService service.UserService
	logger      *logrus.Logger
	config      *configs.Config
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userService service.UserService, logger *logrus.Logger, config *configs.Config) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
		config:      config,
	}
}

// Register handles user registration
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	
	// Parse request body
	var userReg models.UserRegistration
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&userReg); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	defer r.Body.Close()
	
	// Register the user
	userID, err := h.userService.Register(r.Context(), &userReg)
	if err != nil {
		h.logger.Warnf("Failed to register user: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusCreated, "user registered successfully", map[string]interface{}{
		"user_id": userID,
	})
}

// Login handles user login
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	
	// Parse request body
	var loginReq models.UserLogin
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&loginReq); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	defer r.Body.Close()
	
	// Authenticate the user
	tokenResponse, err := h.userService.Login(r.Context(), &loginReq)
	if err != nil {
		h.logger.Warnf("Failed to login user: %v", err)
		utils.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	
	// Return success response with token
	utils.RespondWithSuccess(w, http.StatusOK, "login successful", tokenResponse)
}

// GetUser handles fetching user information
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Get user details
	user, err := h.userService.GetByID(r.Context(), userID)
	if err != nil {
		h.logger.Warnf("Failed to get user: %v", err)
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to get user details")
		return
	}
	
	// Return success response with user details
	utils.RespondWithSuccess(w, http.StatusOK, "user details retrieved successfully", user)
}

// UpdateUser handles updating user information
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Only allow PUT requests
	if r.Method != http.MethodPut {
		utils.RespondWithError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	
	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		utils.RespondWithError(w, http.StatusInternalServerError, "user ID not found in context")
		return
	}
	
	// Parse request body
	var user models.User
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&user); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	defer r.Body.Close()
	
	// Ensure user ID in the request matches the authenticated user ID
	user.ID = userID
	
	// Update the user
	err := h.userService.Update(r.Context(), &user)
	if err != nil {
		h.logger.Warnf("Failed to update user: %v", err)
		utils.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	
	// Return success response
	utils.RespondWithSuccess(w, http.StatusOK, "user updated successfully", nil)
}