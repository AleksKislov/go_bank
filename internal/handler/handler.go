package handler

import (
	"github.com/sirupsen/logrus"

	"banking-service/configs"
	"banking-service/internal/service"
)

// Dependencies contains handler dependencies
type Dependencies struct {
	Services *service.Service
	Logger   *logrus.Logger
	Config   *configs.Config
}

// Handler contains all HTTP handlers for the application
type Handler struct {
	User       *UserHandler
	Account    *AccountHandler
	Card       *CardHandler
	Transaction *TransactionHandler
	Credit     *CreditHandler
	Analytics  *AnalyticsHandler
}

// NewHandler creates a new Handler with all subhandlers
func NewHandler(deps Dependencies) *Handler {
	return &Handler{
		User:       NewUserHandler(deps.Services.User, deps.Logger, deps.Config),
		Account:    NewAccountHandler(deps.Services.Account, deps.Logger, deps.Config),
		Card:       NewCardHandler(deps.Services.Card, deps.Logger, deps.Config),
		Transaction: NewTransactionHandler(deps.Services.Transaction, deps.Logger, deps.Config),
		Credit:     NewCreditHandler(deps.Services.Credit, deps.Logger, deps.Config),
		Analytics:  NewAnalyticsHandler(deps.Services.Analytics, deps.Logger, deps.Config),
	}
}