package service

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"banking-service/configs"
	"banking-service/internal/models"
	"banking-service/internal/repository"
)

// UserService defines methods for user service
type UserService interface {
	Register(ctx context.Context, user *models.UserRegistration) (int, error)
	Login(ctx context.Context, login *models.UserLogin) (*models.TokenResponse, error)
	GetByID(ctx context.Context, id int) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
}

// AccountService defines methods for account service
type AccountService interface {
	Create(ctx context.Context, account *models.AccountCreate) (int, error)
	GetByID(ctx context.Context, id int, userID int) (*models.Account, error)
	GetByUserID(ctx context.Context, userID int) ([]*models.Account, error)
	Deposit(ctx context.Context, accountID int, userID int, deposit *models.DepositRequest) (int, error)
	Withdraw(ctx context.Context, accountID int, userID int, withdrawal *models.WithdrawalRequest) (int, error)
	Update(ctx context.Context, account *models.Account, userID int) error
	Delete(ctx context.Context, id int, userID int) error
}

// CardService defines methods for card service
type CardService interface {
	Create(ctx context.Context, card *models.CardCreate, userID int) (int, error)
	GetByID(ctx context.Context, id int, userID int) (*models.CardResponse, error)
	GetByUserID(ctx context.Context, userID int) ([]*models.CardResponse, error)
	GetByAccountID(ctx context.Context, accountID int, userID int) ([]*models.CardResponse, error)
	Update(ctx context.Context, card *models.Card, userID int) error
	Delete(ctx context.Context, id int, userID int) error
}

// TransactionService defines methods for transaction service
type TransactionService interface {
	Transfer(ctx context.Context, transfer *models.TransferRequest, userID int) (int, error)
	Pay(ctx context.Context, payment *models.PaymentRequest, userID int) (int, error)
	GetByID(ctx context.Context, id int, userID int) (*models.Transaction, error)
	GetByUserID(ctx context.Context, userID int) ([]*models.Transaction, error)
	GetByAccountID(ctx context.Context, accountID int, userID int) ([]*models.Transaction, error)
	GetByDateRange(ctx context.Context, userID int, startDate, endDate time.Time) ([]*models.Transaction, error)
}

// CreditService defines methods for credit service
type CreditService interface {
	Create(ctx context.Context, credit *models.CreditRequest) (int, error)
	GetByID(ctx context.Context, id int, userID int) (*models.Credit, error)
	GetByUserID(ctx context.Context, userID int) ([]*models.Credit, error)
	GetSchedule(ctx context.Context, creditID int, userID int) ([]*models.PaymentScheduleResponse, *models.PaymentScheduleSummary, error)
	ProcessPayments(ctx context.Context) error
	GetKeyRate(ctx context.Context) (float64, error)
}

// AnalyticsService defines methods for analytics service
type AnalyticsService interface {
	GetStatistics(ctx context.Context, userID int, period string) (map[string]interface{}, error)
	PredictBalance(ctx context.Context, accountID int, userID int, days int) (map[string]interface{}, error)
	GetCreditAnalytics(ctx context.Context, userID int) (map[string]interface{}, error)
}

// EmailService defines methods for email service
type EmailService interface {
	SendTransactionNotification(ctx context.Context, userID int, transaction *models.Transaction) error
	SendPaymentReminder(ctx context.Context, userID int, payment *models.PaymentSchedule, credit *models.Credit) error
	SendCreditApproval(ctx context.Context, userID int, credit *models.Credit) error
}

// Dependencies contains dependencies for services
type Dependencies struct {
	Repos  *repository.Repository
	Logger *logrus.Logger
	Config *configs.Config
}

// Service is a composition of all services
type Service struct {
	User       UserService
	Account    AccountService
	Card       CardService
	Transaction TransactionService
	Credit     CreditService
	Analytics  AnalyticsService
	Email      EmailService
}

// NewService creates a new service with all sub-services
func NewService(deps Dependencies) *Service {
	return &Service{
		User:       NewUserService(deps),
		Account:    NewAccountService(deps),
		Card:       NewCardService(deps),
		Transaction: NewTransactionService(deps),
		Credit:     NewCreditService(deps),
		Analytics:  NewAnalyticsService(deps),
		Email:      NewEmailService(deps),
	}
}