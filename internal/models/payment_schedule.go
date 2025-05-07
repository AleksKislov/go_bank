package models

import (
	"time"
)

// PaymentScheduleResponse represents a payment schedule for API responses
type PaymentScheduleResponse struct {
	ID             int           `json:"id"`
	CreditID       int           `json:"credit_id"`
	PaymentNumber  int           `json:"payment_number"`
	PaymentDate    time.Time     `json:"payment_date"`
	PrincipalAmount float64      `json:"principal_amount"`
	InterestAmount float64       `json:"interest_amount"`
	TotalAmount    float64       `json:"total_amount"`
	Status         PaymentStatus `json:"status"`
	IsOverdue      bool          `json:"is_overdue"`
	PenaltyAmount  float64       `json:"penalty_amount,omitempty"`
}

// PaymentScheduleSummary represents summary statistics for a payment schedule
type PaymentScheduleSummary struct {
	TotalPayments      int     `json:"total_payments"`
	TotalPrincipal     float64 `json:"total_principal"`
	TotalInterest      float64 `json:"total_interest"`
	TotalAmount        float64 `json:"total_amount"`
	RemainingPayments  int     `json:"remaining_payments"`
	RemainingPrincipal float64 `json:"remaining_principal"`
	RemainingInterest  float64 `json:"remaining_interest"`
	RemainingAmount    float64 `json:"remaining_amount"`
	PaidPayments       int     `json:"paid_payments"`
	PaidPrincipal      float64 `json:"paid_principal"`
	PaidInterest       float64 `json:"paid_interest"`
	PaidAmount         float64 `json:"paid_amount"`
	OverduePayments    int     `json:"overdue_payments"`
	OverduePrincipal   float64 `json:"overdue_principal"`
	OverdueInterest    float64 `json:"overdue_interest"`
	OverdueAmount      float64 `json:"overdue_amount"`
	TotalPenalties     float64 `json:"total_penalties"`
}

// ToPaymentScheduleResponse converts PaymentSchedule to PaymentScheduleResponse
func (p *PaymentSchedule) ToPaymentScheduleResponse(paymentNumber int) *PaymentScheduleResponse {
	return &PaymentScheduleResponse{
		ID:              p.ID,
		CreditID:        p.CreditID,
		PaymentNumber:   paymentNumber,
		PaymentDate:     p.PaymentDate,
		PrincipalAmount: p.PrincipalAmount,
		InterestAmount:  p.InterestAmount,
		TotalAmount:     p.TotalAmount,
		Status:          p.Status,
		IsOverdue:       p.IsOverdue,
		PenaltyAmount:   p.PenaltyAmount,
	}
}

// CalculatePaymentScheduleSummary calculates summary statistics for a payment schedule
func CalculatePaymentScheduleSummary(schedules []*PaymentSchedule) *PaymentScheduleSummary {
	summary := &PaymentScheduleSummary{}
	
	summary.TotalPayments = len(schedules)
	
	for _, payment := range schedules {
		summary.TotalPrincipal += payment.PrincipalAmount
		summary.TotalInterest += payment.InterestAmount
		summary.TotalAmount += payment.TotalAmount
		summary.TotalPenalties += payment.PenaltyAmount
		
		switch payment.Status {
		case PaymentStatusPaid:
			summary.PaidPayments++
			summary.PaidPrincipal += payment.PrincipalAmount
			summary.PaidInterest += payment.InterestAmount
			summary.PaidAmount += payment.TotalAmount
		case PaymentStatusPending:
			summary.RemainingPayments++
			summary.RemainingPrincipal += payment.PrincipalAmount
			summary.RemainingInterest += payment.InterestAmount
			summary.RemainingAmount += payment.TotalAmount
		case PaymentStatusOverdue:
			summary.OverduePayments++
			summary.OverduePrincipal += payment.PrincipalAmount
			summary.OverdueInterest += payment.InterestAmount
			summary.OverdueAmount += payment.TotalAmount
		}
	}
	
	return summary
}

// UpdateScheduleStatus updates the status of a payment schedule item based on the current date
func UpdateScheduleStatus(schedule *PaymentSchedule) {
	now := time.Now()
	
	// Check if payment is overdue
	if schedule.Status == PaymentStatusPending && now.After(schedule.PaymentDate) {
		schedule.IsOverdue = true
		schedule.Status = PaymentStatusOverdue
		
		// Calculate number of days overdue
		daysOverdue := int(now.Sub(schedule.PaymentDate).Hours() / 24)
		
		// Apply penalty (10% of total payment) if overdue more than 1 day
		if daysOverdue > 1 {
			schedule.PenaltyAmount = roundToTwoDecimal(schedule.TotalAmount * 0.1)
		}
	}
}