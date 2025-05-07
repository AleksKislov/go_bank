package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"banking-service/configs"
	"banking-service/internal/models"
	"banking-service/internal/repository"
)

// AnalyticsSvc is an implementation of the service.AnalyticsService interface
type AnalyticsSvc struct {
	repos  *repository.Repository
	logger *logrus.Logger
	config *configs.Config
}

// NewAnalyticsService creates a new AnalyticsSvc
func NewAnalyticsService(deps Dependencies) *AnalyticsSvc {
	return &AnalyticsSvc{
		repos:  deps.Repos,
		logger: deps.Logger,
		config: deps.Config,
	}
}

// GetStatistics gets financial statistics for a user
func (s *AnalyticsSvc) GetStatistics(ctx context.Context, userID int, period string) (map[string]interface{}, error) {
	// Define time range based on period
	var startDate, endDate time.Time
	now := time.Now()
	
	switch period {
	case "week":
		startDate = now.AddDate(0, 0, -7)
	case "month":
		startDate = now.AddDate(0, -1, 0)
	case "quarter":
		startDate = now.AddDate(0, -3, 0)
	case "year":
		startDate = now.AddDate(-1, 0, 0)
	default:
		// Default to month
		period = "month"
		startDate = now.AddDate(0, -1, 0)
	}
	
	endDate = now
	
	// Get transactions for the specified period
	transactions, err := s.repos.Transaction.GetByDateRange(ctx, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	
	// Get accounts for the user
	accounts, err := s.repos.Account.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}
	
	// Get credits for the user
	credits, err := s.repos.Credit.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credits: %w", err)
	}
	
	// Calculate statistics
	stats := calculateStatistics(transactions, accounts, credits)
	
	// Add period info
	stats["period"] = period
	stats["start_date"] = startDate.Format("2006-01-02")
	stats["end_date"] = endDate.Format("2006-01-02")
	
	s.logger.Infof("Generated statistics for user %d for period: %s", userID, period)
	
	return stats, nil
}

// PredictBalance predicts account balance for future days
func (s *AnalyticsSvc) PredictBalance(ctx context.Context, accountID int, userID int, days int) (map[string]interface{}, error) {
	// Verify account ownership
	account, err := s.repos.Account.GetByID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}
	
	if account.UserID != userID {
		return nil, errors.New("access denied: account belongs to another user")
	}
	
	// Set reasonable limit for prediction days
	if days <= 0 {
		days = 30 // Default to 30 days
	} else if days > 365 {
		days = 365 // Max 1 year
	}
	
	// Get upcoming credit payments for this account
	var creditPayments []*models.PaymentSchedule
	credits, err := s.repos.Credit.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credits: %w", err)
	}
	
	endDate := time.Now().AddDate(0, 0, days)
	
	for _, credit := range credits {
		if credit.Status != models.CreditStatusActive {
			continue
		}
		
		payments, err := s.repos.PaymentSchedule.GetByCreditID(ctx, credit.ID)
		if err != nil {
			s.logger.Warnf("Failed to get payment schedule for credit %d: %v", credit.ID, err)
			continue
		}
		
		for _, payment := range payments {
			if payment.Status == models.PaymentStatusPending && payment.PaymentDate.Before(endDate) {
				creditPayments = append(creditPayments, payment)
			}
		}
	}
	
	// Get historical transactions for trend analysis
	startDate := time.Now().AddDate(0, -3, 0) // Last 3 months
	transactions, err := s.repos.Transaction.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}
	
	// Filter transactions by date
	var recentTransactions []*models.Transaction
	for _, tx := range transactions {
		if tx.TransactionDate.After(startDate) {
			recentTransactions = append(recentTransactions, tx)
		}
	}
	
	// Calculate prediction
	prediction := predictAccountBalance(account, recentTransactions, creditPayments, days)
	
	s.logger.Infof("Generated balance prediction for account %d for %d days", accountID, days)
	
	return prediction, nil
}

// GetCreditAnalytics gets credit analysis for a user
func (s *AnalyticsSvc) GetCreditAnalytics(ctx context.Context, userID int) (map[string]interface{}, error) {
	// Get credits for the user
	credits, err := s.repos.Credit.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credits: %w", err)
	}
	
	// Get payment schedules for all credits
	var allSchedules []*models.PaymentSchedule
	var creditSummaries []map[string]interface{}
	
	totalDebt := 0.0
	totalPaidInterest := 0.0
	totalOverduePayments := 0.0
	totalMonthlyPayment := 0.0
	
	for _, credit := range credits {
		if credit.Status != models.CreditStatusActive && credit.Status != models.CreditStatusOverdue {
			continue
		}
		
		schedules, err := s.repos.PaymentSchedule.GetByCreditID(ctx, credit.ID)
		if err != nil {
			s.logger.Warnf("Failed to get payment schedule for credit %d: %v", credit.ID, err)
			continue
		}
		
		allSchedules = append(allSchedules, schedules...)
		
		// Calculate summary for this credit
		summary := models.CalculatePaymentScheduleSummary(schedules)
		
		creditSummary := map[string]interface{}{
			"credit_id":            credit.ID,
			"amount":               credit.Amount,
			"interest_rate":        credit.InterestRate,
			"term_months":          credit.TermMonths,
			"monthly_payment":      credit.MonthlyPayment,
			"start_date":           credit.StartDate.Format("2006-01-02"),
			"end_date":             credit.EndDate.Format("2006-01-02"),
			"status":               credit.Status,
			"remaining_principal":  summary.RemainingPrincipal,
			"remaining_interest":   summary.RemainingInterest,
			"paid_principal":       summary.PaidPrincipal,
			"paid_interest":        summary.PaidInterest,
			"overdue_amount":       summary.OverdueAmount,
			"total_penalties":      summary.TotalPenalties,
			"remaining_payments":   summary.RemainingPayments,
		}
		
		creditSummaries = append(creditSummaries, creditSummary)
		
		totalDebt += summary.RemainingPrincipal + summary.RemainingInterest
		totalPaidInterest += summary.PaidInterest
		totalOverduePayments += summary.OverdueAmount + summary.TotalPenalties
		totalMonthlyPayment += credit.MonthlyPayment
	}
	
	// Calculate debt to income ratio (if we have income data)
	debtToIncomeRatio := 0.0
	monthlyIncome := estimateMonthlyIncome(ctx, s.repos, userID)
	
	if monthlyIncome > 0 {
		debtToIncomeRatio = totalMonthlyPayment / monthlyIncome
	}
	
	// Prepare credit analysis
	creditAnalysis := map[string]interface{}{
		"total_credits":          len(credits),
		"active_credits":         len(creditSummaries),
		"total_debt":             totalDebt,
		"total_paid_interest":    totalPaidInterest,
		"total_overdue_payments": totalOverduePayments,
		"total_monthly_payment":  totalMonthlyPayment,
		"debt_to_income_ratio":   debtToIncomeRatio,
		"credit_summaries":       creditSummaries,
	}
	
	s.logger.Infof("Generated credit analytics for user %d", userID)
	
	return creditAnalysis, nil
}

// Helper function to calculate statistics
func calculateStatistics(transactions []*models.Transaction, accounts []*models.Account, credits []*models.Credit) map[string]interface{} {
	totalBalance := 0.0
	totalDebt := 0.0
	totalIncome := 0.0
	totalExpenses := 0.0
	
	// Calculate totals from accounts and credits
	for _, account := range accounts {
		if account.AccountType != models.AccountTypeCredit {
			totalBalance += account.Balance
		}
	}
	
	for _, credit := range credits {
		if credit.Status == models.CreditStatusActive || credit.Status == models.CreditStatusOverdue {
			totalDebt += credit.Amount
		}
	}
	
	// Categorize transactions
	categoryIncome := make(map[string]float64)
	categoryExpense := make(map[string]float64)
	
	for _, tx := range transactions {
		category := categorizeTransaction(tx)
		
		if tx.TransactionType == models.TransactionTypeDeposit {
			totalIncome += tx.Amount
			categoryIncome[category] += tx.Amount
		} else if tx.TransactionType == models.TransactionTypeWithdrawal || 
			tx.TransactionType == models.TransactionTypePayment {
			totalExpenses += tx.Amount
			categoryExpense[category] += tx.Amount
		}
	}
	
	// Prepare stats response
	stats := map[string]interface{}{
		"total_balance":     totalBalance,
		"total_debt":        totalDebt,
		"total_income":      totalIncome,
		"total_expenses":    totalExpenses,
		"net_flow":          totalIncome - totalExpenses,
		"category_income":   categoryIncome,
		"category_expenses": categoryExpense,
	}
	
	return stats
}

// Helper function to predict account balance
func predictAccountBalance(account *models.Account, transactions []*models.Transaction, creditPayments []*models.PaymentSchedule, days int) map[string]interface{} {
	now := time.Now()
	
	// Prepare daily predictions
	dailyPredictions := make([]map[string]interface{}, days+1)
	currentBalance := account.Balance
	
	// Initialize first day (today)
	dailyPredictions[0] = map[string]interface{}{
		"date":    now.Format("2006-01-02"),
		"balance": currentBalance,
		"events":  []string{},
	}
	
	// Calculate average daily income/expense based on historical data
	var regularIncome, regularExpense float64
	if len(transactions) > 0 {
		var totalIncome, totalExpense float64
		var incomeCount, expenseCount int
		
		for _, tx := range transactions {
			if tx.TransactionType == models.TransactionTypeDeposit {
				totalIncome += tx.Amount
				incomeCount++
			} else if tx.TransactionType == models.TransactionTypeWithdrawal || 
				tx.TransactionType == models.TransactionTypePayment {
				totalExpense += tx.Amount
				expenseCount++
			}
		}
		
		// Calculate daily averages
		daysInPeriod := now.Sub(transactions[len(transactions)-1].TransactionDate).Hours() / 24
		if daysInPeriod < 1 {
			daysInPeriod = 1
		}
		
		regularIncome = totalIncome / daysInPeriod
		regularExpense = totalExpense / daysInPeriod
	}
	
	// Project balance for each day
	for day := 1; day <= days; day++ {
		date := now.AddDate(0, 0, day)
		events := []string{}
		
		// Apply regular income/expense trends
		dailyIncome := regularIncome
		dailyExpense := regularExpense
		
		// Apply scheduled credit payments
		for _, payment := range creditPayments {
			if isSameDay(payment.PaymentDate, date) {
				currentBalance -= payment.TotalAmount
				events = append(events, fmt.Sprintf("Credit payment: -%.2f", payment.TotalAmount))
			}
		}
		
		// Apply daily trend
		currentBalance += dailyIncome - dailyExpense
		
		// Salary deposit simulation (assuming monthly salary on 10th)
		if date.Day() == 10 {
			// Estimate a salary deposit based on previous income
			estimatedSalary := regularIncome * 30 * 0.7 // 70% of monthly income as salary
			if estimatedSalary > 0 {
				currentBalance += estimatedSalary
				events = append(events, fmt.Sprintf("Estimated salary: +%.2f", estimatedSalary))
			}
		}
		
		// Store daily prediction
		dailyPredictions[day] = map[string]interface{}{
			"date":    date.Format("2006-01-02"),
			"balance": currentBalance,
			"events":  events,
		}
	}
	
	// Calculate min, max, end balance
	minBalance := account.Balance
	maxBalance := account.Balance
	
	for _, prediction := range dailyPredictions {
		balance := prediction["balance"].(float64)
		if balance < minBalance {
			minBalance = balance
		}
		if balance > maxBalance {
			maxBalance = balance
		}
	}
	
	// Prepare prediction result
	prediction := map[string]interface{}{
		"account_id":      account.ID,
		"current_balance": account.Balance,
		"min_balance":     minBalance,
		"max_balance":     maxBalance,
		"end_balance":     dailyPredictions[days]["balance"],
		"days_predicted":  days,
		"daily_predictions": dailyPredictions,
	}
	
	return prediction
}

// Helper function to check if two dates are the same day
func isSameDay(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// Helper function to categorize a transaction
func categorizeTransaction(tx *models.Transaction) string {
	// Simple keyword-based categorization
	description := tx.Description
	
	if description == "" {
		return "Other"
	}
	
	keywords := map[string]string{
		"salary":     "Salary",
		"wages":      "Salary",
		"rent":       "Housing",
		"mortgage":   "Housing",
		"apartment":  "Housing",
		"grocery":    "Groceries",
		"food":       "Groceries",
		"restaurant": "Dining",
		"cafe":       "Dining",
		"coffee":     "Dining",
		"transport":  "Transportation",
		"taxi":       "Transportation",
		"uber":       "Transportation",
		"bus":        "Transportation",
		"train":      "Transportation",
		"metro":      "Transportation",
		"pharmacy":   "Healthcare",
		"doctor":     "Healthcare",
		"hospital":   "Healthcare",
		"medical":    "Healthcare",
		"utility":    "Utilities",
		"electricity":"Utilities",
		"water":      "Utilities",
		"gas":        "Utilities",
		"internet":   "Utilities",
		"phone":      "Utilities",
		"mobile":     "Utilities",
		"insurance":  "Insurance",
		"credit":     "Credit Payment",
		"loan":       "Credit Payment",
		"interest":   "Credit Payment",
		"fee":        "Bank Fees",
		"transfer":   "Transfer",
	}
	
	for keyword, category := range keywords {
		if strings.Contains(strings.ToLower(description), keyword) {
			return category
		}
	}
	
	return "Other"
}

// Helper function to estimate monthly income
func estimateMonthlyIncome(ctx context.Context, repos *repository.Repository, userID int) float64 {
	// Get transactions for the last 3 months
	now := time.Now()
	startDate := now.AddDate(0, -3, 0)
	
	transactions, err := repos.Transaction.GetByDateRange(ctx, userID, startDate, now)
	if err != nil {
		return 0
	}
	
	// Find deposit transactions that might represent income
	var incomeTransactions []*models.Transaction
	for _, tx := range transactions {
		if tx.TransactionType == models.TransactionTypeDeposit {
			if categorizeTransaction(tx) == "Salary" {
				incomeTransactions = append(incomeTransactions, tx)
			}
		}
	}
	
	// If we have no salary transactions, use all deposits
	if len(incomeTransactions) == 0 {
		for _, tx := range transactions {
			if tx.TransactionType == models.TransactionTypeDeposit {
				incomeTransactions = append(incomeTransactions, tx)
			}
		}
	}
	
	// Calculate average monthly income
	var totalIncome float64
	for _, tx := range incomeTransactions {
		totalIncome += tx.Amount
	}
	
	// Convert to monthly average
	monthsPeriod := 3.0
	if len(incomeTransactions) > 0 {
		return totalIncome / monthsPeriod
	}
	
	return 0
}