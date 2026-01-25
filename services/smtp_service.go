package services

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"

	"member_API/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	maxEmailRetries = 3
)

type SMTPService struct {
	DB *gorm.DB
}

func NewSMTPService(db *gorm.DB) *SMTPService {
	return &SMTPService{DB: db}
}

type EmailRequest struct {
	RecipientEmail string `json:"recipient_email" binding:"required,email"`
	RecipientName  string `json:"recipient_name"`
	Subject        string `json:"subject" binding:"required"`
	Body           string `json:"body" binding:"required"`
}

func (s *SMTPService) SendEmail(memberID uuid.UUID, providerID uuid.UUID, req EmailRequest) error {
	var provider models.NotificationProvider
	if err := s.DB.Where("id = ? AND is_active = ? AND type = ?", providerID, true, models.ProviderSMTP).First(&provider).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("smtp provider not found or inactive")
		}
		return err
	}

	var config models.SMTPConfig
	if err := json.Unmarshal([]byte(provider.Config), &config); err != nil {
		return fmt.Errorf("invalid smtp config: %w", err)
	}

	notifLog := models.NotificationLog{
		MemberID:       memberID,
		ProviderID:     providerID,
		Type:           models.ProviderSMTP,
		RecipientEmail: req.RecipientEmail,
		RecipientName:  req.RecipientName,
		Subject:        req.Subject,
		Body:           req.Body,
		Status:         models.StatusPending,
		Base: models.Base{
			CreationTime: time.Now(),
			CreatorId:    memberID,
			IsDeleted:    false,
		},
	}

	var lastErr error

	for attempt := 0; attempt < maxEmailRetries; attempt++ {
		err := s.sendEmailWithConfig(config, req)
		if err == nil {
			now := time.Now()
			notifLog.Status = models.StatusSent
			notifLog.SentAt = &now
			if err := s.DB.Create(&notifLog).Error; err != nil {
				log.Printf("Warning: Failed to create success log: %v", err)
			}
			return nil
		}
		lastErr = err
		if attempt < maxEmailRetries-1 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	notifLog.Status = models.StatusFailed
	notifLog.ErrorMsg = lastErr.Error()
	if err := s.DB.Create(&notifLog).Error; err != nil {
		log.Printf("Warning: Failed to create failure log: %v", err)
	}

	return fmt.Errorf("failed after %d retries: %w", maxEmailRetries, lastErr)
}

func (s *SMTPService) sendEmailWithConfig(config models.SMTPConfig, req EmailRequest) error {
	// Validate config
	if config.Host == "" || config.Port == 0 || config.Username == "" || config.Password == "" || config.From == "" {
		return errors.New("invalid smtp config: missing required fields")
	}

	from := config.From
	to := []string{req.RecipientEmail}
	
	// Build headers in fixed order for consistency
	headerLines := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", req.RecipientEmail),
		fmt.Sprintf("Subject: %s", req.Subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}
	
	message := strings.Join(headerLines, "\r\n") + "\r\n\r\n" + req.Body

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	if config.UseTLS {
		return s.sendMailTLS(addr, auth, from, to, []byte(message))
	}

	return smtp.SendMail(addr, auth, from, to, []byte(message))
}

func (s *SMTPService) sendMailTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	host := strings.Split(addr, ":")[0]

	// Connect to SMTP server without TLS
	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := client.Close(); cerr != nil {
			log.Printf("Warning: Failed to close SMTP client: %v", cerr)
		}
	}()

	// Start TLS for secure communication (STARTTLS)
	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	if err = client.StartTLS(tlsConfig); err != nil {
		return err
	}

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}

	if err = client.Mail(from); err != nil {
		return err
	}

	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return err
		}
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}
