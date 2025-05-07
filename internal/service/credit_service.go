package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/sirupsen/logrus"

	"banking-service/configs"
	"banking-service/internal/models"
	"banking-service/internal/repository"
)

// CBRResponse represents the XML response from Central Bank of Russia
type CBRResponse struct {
	XMLName xml.Name `xml:"envelope"`
	Body    struct {
		XMLName      xml.Name `xml:"Body"`
		GetRateResp  struct {
			XMLName xml.Name `xml:"GetCursOnDateXMLResponse"`
			Result  struct {
				XMLName xml.Name `xml:"GetCursOnDateXMLResult"`
				Rates   string   `xml:",innerxml"`
			}
		}
	}
}

// CreditSvc is an implementation of the service.CreditService interface
type CreditSvc struct {
	repos  *repository.Repository
	logger *logrus.Logger
	config *configs.Config
	email  EmailService
}

// NewCreditService creates a new CreditSvc
func NewCreditService(deps Dependencies) *CreditSvc {
	return &CreditSvc{
		repos:  deps.Repos,
		logger: deps.Logger,
		config: deps.Config,
		email:  NewEmailService(deps),
	}
}

// Create creates a new credit
func (s *CreditSvc) Create(ctx context.Context, creditReq *models.CreditRequest) (int, error) {
	// Validate credit request
	if err := creditReq.ValidateCreditRequest(); err != nil {
		return 0, fmt.Errorf("invalid credit request: %w", err)
	}
	
	// Check if user exists
	user, err := s.repos.User.GetByID(ctx, creditReq.UserID)
	if err != nil {
		return 0, fmt.Errorf("user not found: %w", err)
	}
	
	// Get base interest rate from Central Bank
	baseRate, err := s.GetKeyRate(ctx)
	if err != nil {
		s.logger.Warnf("Failed to get base interest rate: %v. Using default rate of 7%%.", err)
		baseRate = 7.0 // Default rate if CBR API fails
	}
	
	// Start a transaction
	tx, err := s.repos.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	
	// Create a credit account
	creditAccount := &models.Account{
		UserID:        creditReq.UserID,
		AccountNumber: models.GenerateAccountNumber(),
		Balance:       0,
		Currency:      models.CurrencyRUB,
		AccountType:   models.AccountTypeCredit,
		IsActive:      true,
	}
	
	accountID, err := s.repos.Account.Create(ctx, creditAccount)
	if err != nil {
		return 0, fmt.Errorf("failed to create credit account: %w", err)
	}
	
	// Create the credit
	credit := creditReq.ToCredit(accountID, baseRate)
	
	creditID, err := s.repos.Credit.Create(ctx, credit)
	if err != nil {
		return 0, fmt.Errorf("failed to create credit: %w", err)
	}
	
	// Generate payment schedule
	credit.ID = creditID
	schedule := models.GeneratePaymentSchedule(credit)
	
	// Store payment schedule
	err = s.repos.PaymentSchedule.CreateBatch(ctx, schedule)
	if err != nil {
		return 0, fmt.Errorf("failed to create payment schedule: %w", err)
	}
	
	// Add loan amount to credit account
	err = s.repos.Account.UpdateBalance(ctx, accountID, creditReq.Amount)
	if err != nil {
		return 0, fmt.Errorf("failed to update credit account balance: %w", err)
	}
	
	// Create a deposit transaction for the loan
	depositTransaction := &models.Transaction{
		TransactionType:      models.TransactionTypeDeposit,
		DestinationAccountID: &accountID,
		Amount:               creditReq.Amount,
		Currency:             models.CurrencyRUB,
		Description:          fmt.Sprintf("Credit #%d issued", creditID),
		Status:               models.TransactionStatusCompleted,
		TransactionDate:      time.Now(),
	}
	
	_, err = s.repos.Transaction.Create(ctx, depositTransaction)
	if err != nil {
		return 0, fmt.Errorf("failed to create deposit transaction: %w", err)
	}
	
	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	s.logger.Infof("Credit created: %d for user: %d, amount: %f, term: %d months, rate: %f%%",
		creditID, creditReq.UserID, creditReq.Amount, creditReq.TermMonths, credit.InterestRate)
	
	// Send email notification
	go func() {
		ctx := context.Background()
		err := s.email.SendCreditApproval(ctx, user.ID, credit)
		if err != nil {
			s.logger.Warnf("Failed to send credit approval notification: %v", err)
		}
	}()
	
	return creditID, nil
}

// GetByID gets a credit by ID and verifies ownership
func (s *CreditSvc) GetByID(ctx context.Context, id int, userID int) (*models.Credit, error) {
	credit, err := s.repos.Credit.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get credit: %w", err)
	}
	
	if credit.UserID != userID {
		return nil, errors.New("access denied: credit belongs to another user")
	}
	
	return credit, nil
}

// GetByUserID gets all credits for a user
func (s *CreditSvc) GetByUserID(ctx context.Context, userID int) ([]*models.Credit, error) {
	credits, err := s.repos.Credit.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get credits: %w", err)
	}
	
	return credits, nil
}

// GetSchedule gets the payment schedule for a credit and verifies ownership
func (s *CreditSvc) GetSchedule(ctx context.Context, creditID int, userID int) ([]*models.PaymentScheduleResponse, *models.PaymentScheduleSummary, error) {
	// Verify credit ownership
	credit, err := s.GetByID(ctx, creditID, userID)
	if err != nil {
		return nil, nil, err
	}
	
	// Get payment schedule
	schedules, err := s.repos.PaymentSchedule.GetByCreditID(ctx, creditID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get payment schedule: %w", err)
	}
	
	// Check for any overdue payments and update them
	updated := false
	for _, schedule := range schedules {
		if schedule.Status == models.PaymentStatusPending {
			prevStatus := schedule.Status
			models.UpdateScheduleStatus(schedule)
			
			if prevStatus != schedule.Status {
				err := s.repos.PaymentSchedule.Update(ctx, schedule)
				if err != nil {
					s.logger.Warnf("Failed to update payment schedule status: %v", err)
				} else {
					updated = true
				}
			}
		}
	}
	
	// If any payments were updated to overdue, update the credit status as well
	if updated {
		credit.Status = models.CreditStatusOverdue
		err := s.repos.Credit.Update(ctx, credit)
		if err != nil {
			s.logger.Warnf("Failed to update credit status: %v", err)
		}
	}
	
	// Convert to response objects
	var responses []*models.PaymentScheduleResponse
	for i, schedule := range schedules {
		response := schedule.ToPaymentScheduleResponse(i + 1)
		responses = append(responses, response)
	}
	
	// Calculate summary
	summary := models.CalculatePaymentScheduleSummary(schedules)
	
	return responses, summary, nil
}

// ProcessPayments processes all pending payments that are due today
func (s *CreditSvc) ProcessPayments(ctx context.Context) error {
	today := time.Now()
	s.logger.Infof("Processing payments for date: %s", today.Format("2006-01-02"))
	
	// Get all pending payments due today or earlier
	pendingPayments, err := s.repos.PaymentSchedule.GetPendingPayments(ctx, today)
	if err != nil {
		return fmt.Errorf("failed to get pending payments: %w", err)
	}
	
	s.logger.Infof("Found %d pending payments to process", len(pendingPayments))
	
	for _, payment := range pendingPayments {
		// Get the credit for this payment
		credit, err := s.repos.Credit.GetByID(ctx, payment.CreditID)
		if err != nil {
			s.logger.Warnf("Failed to get credit %d for payment %d: %v", payment.CreditID, payment.ID, err)
			continue
		}
		
		// Get the account for this credit
		account, err := s.repos.Account.GetByID(ctx, credit.AccountID)
		if err != nil {
			s.logger.Warnf("Failed to get account %d for credit %d: %v", credit.AccountID, credit.ID, err)
			continue
		}
		
		// Check if payment is overdue and apply penalty if needed
		models.UpdateScheduleStatus(payment)
		
		// Try to process the payment
		totalAmount := payment.TotalAmount
		if payment.IsOverdue {
			totalAmount += payment.PenaltyAmount
		}
		
		// Start a transaction
		tx, err := s.repos.DB.BeginTx(ctx, nil)
		if err != nil {
			s.logger.Warnf("Failed to begin transaction for payment %d: %v", payment.ID, err)
			continue
		}
		
		// Deduct payment from account
		err = s.repos.Account.UpdateBalance(ctx, account.ID, -totalAmount)
		if err != nil {
			s.logger.Warnf("Failed to update account balance for payment %d: %v", payment.ID, err)
			tx.Rollback()
			
			// If insufficient funds, mark as overdue
			if strings.Contains(err.Error(), "insufficient funds") {
				payment.Status = models.PaymentStatusOverdue
				payment.IsOverdue = true
				
				if payment.PenaltyAmount == 0 {
					payment.PenaltyAmount = payment.TotalAmount * 0.1 // 10% penalty
				}
				
				err = s.repos.PaymentSchedule.Update(ctx, payment)
				if err != nil {
					s.logger.Warnf("Failed to update payment status to overdue: %v", err)
				}
				
				// Update credit status to overdue
				credit.Status = models.CreditStatusOverdue
				err = s.repos.Credit.Update(ctx, credit)
				if err != nil {
					s.logger.Warnf("Failed to update credit status to overdue: %v", err)
				}
				
				// Send reminder email
				go func(userID int, payment *models.PaymentSchedule, credit *models.Credit) {
					ctx := context.Background()
					err := s.email.SendPaymentReminder(ctx, userID, payment, credit)
					if err != nil {
						s.logger.Warnf("Failed to send payment reminder: %v", err)
					}
				}(credit.UserID, payment, credit)
			}
			
			continue
		}
		
		// Create a payment transaction
		paymentTransaction := &models.Transaction{
			TransactionType:  models.TransactionTypePayment,
			SourceAccountID: &account.ID,
			Amount:          totalAmount,
			Currency:        models.CurrencyRUB,
			Description:     fmt.Sprintf("Credit payment for credit #%d", credit.ID),
			Status:          models.TransactionStatusCompleted,
			TransactionDate: time.Now(),
		}
		
		_, err = s.repos.Transaction.Create(ctx, paymentTransaction)
		if err != nil {
			s.logger.Warnf("Failed to create payment transaction: %v", err)
			tx.Rollback()
			continue
		}
		
		// Update payment status
		payment.Status = models.PaymentStatusPaid
		err = s.repos.PaymentSchedule.Update(ctx, payment)
		if err != nil {
			s.logger.Warnf("Failed to update payment status: %v", err)
			tx.Rollback()
			continue
		}
		
		// Commit the transaction
		err = tx.Commit()
		if err != nil {
			s.logger.Warnf("Failed to commit transaction: %v", err)
			continue
		}
		
		s.logger.Infof("Processed payment %d for credit %d, amount: %f", payment.ID, credit.ID, totalAmount)
	}
	
	return nil
}

// GetKeyRate gets the key interest rate from Central Bank of Russia
func (s *CreditSvc) GetKeyRate(ctx context.Context) (float64, error) {
	// Prepare SOAP request
	soapEnvelope := `
	<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:web="http://web.cbr.ru/">
		<soapenv:Header/>
		<soapenv:Body>
			<web:GetCursOnDateXML>
				<web:On_date>` + time.Now().Format("2006-01-02") + `</web:On_date>
			</web:GetCursOnDateXML>
		</soapenv:Body>
	</soapenv:Envelope>`
	
	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", s.config.CBR.APIURL, strings.NewReader(soapEnvelope))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", "http://web.cbr.ru/GetCursOnDateXML")
	
	// Send the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}
	
	// Parse the XML response
	var cbrResp CBRResponse
	err = xml.Unmarshal(body, &cbrResp)
	if err != nil {
		return 0, fmt.Errorf("failed to parse XML response: %w", err)
	}
	
	// Use etree to parse the inner XML content
	doc := etree.NewDocument()
	err = doc.ReadFromString(cbrResp.Body.GetRateResp.Result.Rates)
	if err != nil {
		return 0, fmt.Errorf("failed to parse rate data: %w", err)
	}
	
	// Find the key rate element (usually has ID R01010 for CBR key rate)
	keyRateElem := doc.FindElement("//ValCurs/Valute[@ID='R01010']")
	if keyRateElem == nil {
		return 0, errors.New("key rate element not found in response")
	}
	
	// Extract the value
	valueElem := keyRateElem.FindElement("Value")
	if valueElem == nil {
		return 0, errors.New("value element not found in key rate")
	}
	
	// Parse the value string to float (replace comma with dot)
	var keyRate float64
	valueStr := strings.Replace(valueElem.Text(), ",", ".", 1)
	_, err = fmt.Sscanf(valueStr, "%f", &keyRate)
	if err != nil {
		return 0, fmt.Errorf("failed to parse key rate value: %w", err)
	}
	
	s.logger.Infof("Retrieved key rate from CBR: %f%%", keyRate)
	
	return keyRate, nil
}