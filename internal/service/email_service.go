package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"

	"banking-service/configs"
	"banking-service/internal/models"
	"banking-service/internal/repository"
)

// EmailSvc is an implementation of the service.EmailService interface
type EmailSvc struct {
	repos  *repository.Repository
	logger *logrus.Logger
	config *configs.Config
}

// NewEmailService creates a new EmailSvc
func NewEmailService(deps Dependencies) *EmailSvc {
	return &EmailSvc{
		repos:  deps.Repos,
		logger: deps.Logger,
		config: deps.Config,
	}
}

// SendTransactionNotification sends a notification email for a transaction
func (s *EmailSvc) SendTransactionNotification(ctx context.Context, userID int, transaction *models.Transaction) error {
	// Get the user
	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	// Skip if email is empty
	if user.Email == "" {
		return nil
	}
	
	// Prepare transaction details
	var accountID int
	var transactionType string
	var amountStr string
	
	if transaction.TransactionType == models.TransactionTypeDeposit {
		if transaction.DestinationAccountID == nil {
			return fmt.Errorf("deposit transaction has no destination account")
		}
		accountID = *transaction.DestinationAccountID
		transactionType = "Deposit"
		amountStr = fmt.Sprintf("+%.2f %s", transaction.Amount, transaction.Currency)
	} else if transaction.TransactionType == models.TransactionTypeWithdrawal || 
		transaction.TransactionType == models.TransactionTypePayment ||
		transaction.TransactionType == models.TransactionTypeTransfer {
		if transaction.SourceAccountID == nil {
			return fmt.Errorf("withdrawal/payment/transfer transaction has no source account")
		}
		accountID = *transaction.SourceAccountID
		
		if transaction.TransactionType == models.TransactionTypeWithdrawal {
			transactionType = "Withdrawal"
		} else if transaction.TransactionType == models.TransactionTypePayment {
			transactionType = "Payment"
		} else {
			transactionType = "Transfer"
		}
		
		amountStr = fmt.Sprintf("-%.2f %s", transaction.Amount, transaction.Currency)
	} else {
		transactionType = string(transaction.TransactionType)
		amountStr = fmt.Sprintf("%.2f %s", transaction.Amount, transaction.Currency)
	}
	
	// Get account details
	account, err := s.repos.Account.GetByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	
	// Create email content
	subject := fmt.Sprintf("%s Notification: %s", transactionType, amountStr)
	
	body := fmt.Sprintf(`
	<h2>Transaction Notification</h2>
	<p>Dear %s %s,</p>
	
	<p>We are informing you about a recent transaction on your account:</p>
	
	<table style="border-collapse: collapse; width: 100%%;">
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Transaction Type:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Amount:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Account:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Current Balance:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%.2f %s</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Date:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Description:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
		</tr>
	</table>
	
	<p>If you did not authorize this transaction, please contact our support immediately.</p>
	
	<p>Thank you for using our banking services.</p>
	
	<p>
	Best regards,<br>
	Banking Service Team
	</p>
	`,
		user.FirstName, user.LastName,
		transactionType,
		amountStr,
		account.AccountNumber,
		account.Balance, account.Currency,
		transaction.TransactionDate.Format("2006-01-02 15:04:05"),
		transaction.Description,
	)
	
	// Send the email
	err = s.sendEmail(user.Email, subject, body)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	s.logger.Infof("Transaction notification email sent to %s for transaction %d", user.Email, transaction.ID)
	
	return nil
}

// SendPaymentReminder sends a reminder for an upcoming or overdue payment
func (s *EmailSvc) SendPaymentReminder(ctx context.Context, userID int, payment *models.PaymentSchedule, credit *models.Credit) error {
	// Get the user
	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	// Skip if email is empty
	if user.Email == "" {
		return nil
	}
	
	// Get account details
	account, err := s.repos.Account.GetByID(ctx, credit.AccountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	
	// Determine if payment is overdue
	isOverdue := payment.IsOverdue
	
	// Create email content
	var subject string
	if isOverdue {
		subject = fmt.Sprintf("OVERDUE Payment Reminder: Credit #%d", credit.ID)
	} else {
		subject = fmt.Sprintf("Upcoming Payment Reminder: Credit #%d", credit.ID)
	}
	
	// Calculate total amount with penalty if overdue
	totalAmount := payment.TotalAmount
	if isOverdue && payment.PenaltyAmount > 0 {
		totalAmount += payment.PenaltyAmount
	}
	
	var overdueText string
	if isOverdue {
		daysOverdue := int(time.Now().Sub(payment.PaymentDate).Hours() / 24)
		overdueText = fmt.Sprintf(`
		<p style="color: red; font-weight: bold;">
			This payment is OVERDUE by %d days. A penalty of %.2f RUB has been applied.
		</p>
		`, daysOverdue, payment.PenaltyAmount)
	} else {
		daysUntil := int(payment.PaymentDate.Sub(time.Now()).Hours() / 24)
		overdueText = fmt.Sprintf(`
		<p>
			This payment is due in %d days. Please ensure you have sufficient funds in your account.
		</p>
		`, daysUntil)
	}
	
	body := fmt.Sprintf(`
	<h2>Credit Payment Reminder</h2>
	<p>Dear %s %s,</p>
	
	%s
	
	<p>Here are the details of your credit payment:</p>
	
	<table style="border-collapse: collapse; width: 100%%;">
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Credit ID:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%d</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Payment Date:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Principal Amount:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%.2f RUB</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Interest Amount:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%.2f RUB</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Penalty Amount:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%.2f RUB</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Total Amount Due:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%.2f RUB</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Account Number:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Current Account Balance:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%.2f RUB</td>
		</tr>
	</table>
	
	<p>Please ensure you have sufficient funds in your account to cover this payment.</p>
	
	<p>Thank you for using our banking services.</p>
	
	<p>
	Best regards,<br>
	Banking Service Team
	</p>
	`,
		user.FirstName, user.LastName,
		overdueText,
		credit.ID,
		payment.PaymentDate.Format("2006-01-02"),
		payment.PrincipalAmount,
		payment.InterestAmount,
		payment.PenaltyAmount,
		totalAmount,
		account.AccountNumber,
		account.Balance,
	)
	
	// Send the email
	err = s.sendEmail(user.Email, subject, body)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	s.logger.Infof("Payment reminder email sent to %s for credit %d", user.Email, credit.ID)
	
	return nil
}

// SendCreditApproval sends a notification for an approved credit
func (s *EmailSvc) SendCreditApproval(ctx context.Context, userID int, credit *models.Credit) error {
	// Get the user
	user, err := s.repos.User.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	
	// Skip if email is empty
	if user.Email == "" {
		return nil
	}
	
	// Get account details
	account, err := s.repos.Account.GetByID(ctx, credit.AccountID)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}
	
	// Get payment schedule for the first payment
	schedules, err := s.repos.PaymentSchedule.GetByCreditID(ctx, credit.ID)
	if err != nil || len(schedules) == 0 {
		s.logger.Warnf("Failed to get payment schedule for credit %d: %v", credit.ID, err)
	}
	
	var firstPaymentDate string
	if len(schedules) > 0 {
		firstPaymentDate = schedules[0].PaymentDate.Format("2006-01-02")
	} else {
		firstPaymentDate = "See your payment schedule for details"
	}
	
	// Create email content
	subject := fmt.Sprintf("Credit Approved: %.2f RUB", credit.Amount)
	
	body := fmt.Sprintf(`
	<h2>Credit Approval Notification</h2>
	<p>Dear %s %s,</p>
	
	<p>We are pleased to inform you that your credit application has been approved!</p>
	
	<p>Here are the details of your new credit:</p>
	
	<table style="border-collapse: collapse; width: 100%%;">
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Credit ID:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%d</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Amount:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%.2f RUB</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Interest Rate:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%.2f%%</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Term:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%d months</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Monthly Payment:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%.2f RUB</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>First Payment Date:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Credit Account:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%s</td>
		</tr>
		<tr>
			<td style="padding: 8px; border: 1px solid #ddd;"><strong>Current Account Balance:</strong></td>
			<td style="padding: 8px; border: 1px solid #ddd;">%.2f RUB</td>
		</tr>
	</table>
	
	<p>The approved amount has been deposited to your credit account. You can view your payment schedule in your online banking portal.</p>
	
	<p>Thank you for choosing our banking services.</p>
	
	<p>
	Best regards,<br>
	Banking Service Team
	</p>
	`,
		user.FirstName, user.LastName,
		credit.ID,
		credit.Amount,
		credit.InterestRate,
		credit.TermMonths,
		credit.MonthlyPayment,
		firstPaymentDate,
		account.AccountNumber,
		account.Balance,
	)
	
	// Send the email
	err = s.sendEmail(user.Email, subject, body)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	s.logger.Infof("Credit approval email sent to %s for credit %d", user.Email, credit.ID)
	
	return nil
}

// sendEmail sends an email using the SMTP server
func (s *EmailSvc) sendEmail(to, subject, body string) error {
	// Create a new message
	m := gomail.NewMessage()
	m.SetHeader("From", s.config.Email.SenderEmail)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	
	// Create a new dialer
	d := gomail.NewDialer(
		s.config.Email.SMTPHost,
		s.config.Email.SMTPPort,
		s.config.Email.SMTPUser,
		s.config.Email.SMTPPassword,
	)
	
	// Send the email
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	
	return nil
}