package models

import (
	"errors"
	"math"
	"time"
)

// CreditStatus defines the status of a credit
type CreditStatus string

const (
	CreditStatusActive     CreditStatus = "ACTIVE"
	CreditStatusClosed     CreditStatus = "CLOSED"
	CreditStatusOverdue    CreditStatus = "OVERDUE"
	CreditStatusRejected   CreditStatus = "REJECTED"
)

// PaymentStatus defines the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusPaid      PaymentStatus = "PAID"
	PaymentStatusOverdue   PaymentStatus = "OVERDUE"
	PaymentStatusCancelled PaymentStatus = "CANCELLED"
)

// Credit represents a credit issued to a user
type Credit struct {
	ID            int          `json:"id" db:"id"`
	UserID        int          `json:"user_id" db:"user_id"`
	AccountID     int          `json:"account_id" db:"account_id"`
	Amount        float64      `json:"amount" db:"amount"`
	InterestRate  float64      `json:"interest_rate" db:"interest_rate"`
	TermMonths    int          `json:"term_months" db:"term_months"`
	MonthlyPayment float64     `json:"monthly_payment" db:"monthly_payment"`
	StartDate     time.Time    `json:"start_date" db:"start_date"`
	EndDate       time.Time    `json:"end_date" db:"end_date"`
	Status        CreditStatus `json:"status" db:"status"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at" db:"updated_at"`
}

// PaymentSchedule represents a payment schedule for a credit
type PaymentSchedule struct {
	ID             int           `json:"id" db:"id"`
	CreditID       int           `json:"credit_id" db:"credit_id"`
	PaymentDate    time.Time     `json:"payment_date" db:"payment_date"`
	PrincipalAmount float64      `json:"principal_amount" db:"principal_amount"`
	InterestAmount float64       `json:"interest_amount" db:"interest_amount"`
	TotalAmount    float64       `json:"total_amount" db:"total_amount"`
	Status         PaymentStatus `json:"status" db:"status"`
	IsOverdue      bool          `json:"is_overdue" db:"is_overdue"`
	PenaltyAmount  float64       `json:"penalty_amount,omitempty" db:"penalty_amount"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at" db:"updated_at"`
}

// CreditRequest represents a credit application request
type CreditRequest struct {
	UserID      int     `json:"user_id" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
	TermMonths  int     `json:"term_months" binding:"required"`
	InterestRate float64 `json:"interest_rate,omitempty"` // Optional, can be calculated from CBR rate
}

// ValidateCreditRequest validates credit request data
func (c *CreditRequest) ValidateCreditRequest() error {
	if c.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	
	if c.TermMonths < 1 || c.TermMonths > 360 { // Max 30 years
		return errors.New("term must be between 1 and 360 months")
	}
	
	if c.InterestRate < 0 {
		return errors.New("interest rate cannot be negative")
	}
	
	return nil
}

// CalculateMonthlyPayment calculates the monthly payment for an annuity loan
func CalculateMonthlyPayment(principal float64, annualInterestRate float64, termMonths int) float64 {
	// Convert annual interest rate to monthly and from percentage to decimal
	monthlyInterestRate := annualInterestRate / 12 / 100
	
	// Calculate monthly payment using the annuity formula
	if monthlyInterestRate == 0 {
		return principal / float64(termMonths)
	}
	
	return principal * monthlyInterestRate * math.Pow(1+monthlyInterestRate, float64(termMonths)) / 
		(math.Pow(1+monthlyInterestRate, float64(termMonths)) - 1)
}

// GeneratePaymentSchedule generates a payment schedule for a credit
func GeneratePaymentSchedule(credit *Credit) []*PaymentSchedule {
	var schedule []*PaymentSchedule
	
	remainingPrincipal := credit.Amount
	paymentDate := credit.StartDate
	
	monthlyInterestRate := credit.InterestRate / 12 / 100
	
	for i := 0; i < credit.TermMonths; i++ {
		// Calculate interest for this period
		interestAmount := remainingPrincipal * monthlyInterestRate
		
		// Calculate principal for this period
		var principalAmount float64
		if i == credit.TermMonths-1 {
			// Last payment - adjust to ensure the loan is fully paid
			principalAmount = remainingPrincipal
		} else {
			principalAmount = credit.MonthlyPayment - interestAmount
		}
		
		// Ensure we don't have negative principal due to rounding errors
		if principalAmount < 0 {
			principalAmount = 0
		}
		
		// Update remaining principal
		remainingPrincipal -= principalAmount
		
		// Create payment schedule item
		paymentScheduleItem := &PaymentSchedule{
			CreditID:        credit.ID,
			PaymentDate:     paymentDate,
			PrincipalAmount: roundToTwoDecimal(principalAmount),
			InterestAmount:  roundToTwoDecimal(interestAmount),
			TotalAmount:     roundToTwoDecimal(principalAmount + interestAmount),
			Status:          PaymentStatusPending,
		}
		
		schedule = append(schedule, paymentScheduleItem)
		
		// Move to next month
		paymentDate = addOneMonth(paymentDate)
	}
	
	return schedule
}

// Round to two decimal places
func roundToTwoDecimal(value float64) float64 {
	return math.Round(value*100) / 100
}

// Add one month to a date
func addOneMonth(date time.Time) time.Time {
	return date.AddDate(0, 1, 0)
}

// ToCredit converts CreditRequest to Credit
func (c *CreditRequest) ToCredit(accountID int, baseInterestRate float64) *Credit {
	// If interest rate is not provided, use base rate + 5%
	interestRate := c.InterestRate
	if interestRate == 0 {
		interestRate = baseInterestRate + 5.0
	}
	
	startDate := time.Now()
	endDate := startDate.AddDate(0, c.TermMonths, 0)
	
	monthlyPayment := CalculateMonthlyPayment(c.Amount, interestRate, c.TermMonths)
	
	return &Credit{
		UserID:         c.UserID,
		AccountID:      accountID,
		Amount:         c.Amount,
		InterestRate:   interestRate,
		TermMonths:     c.TermMonths,
		MonthlyPayment: roundToTwoDecimal(monthlyPayment),
		StartDate:      startDate,
		EndDate:        endDate,
		Status:         CreditStatusActive,
	}
}