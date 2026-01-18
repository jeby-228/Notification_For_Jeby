package services

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"member_API/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

	log := models.NotificationLog{
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

	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := s.sendEmailWithConfig(config, req)
		if err == nil {
			now := time.Now()
			log.Status = models.StatusSent
			log.SentAt = &now
			s.DB.Create(&log)
			return nil
		}
		lastErr = err
		if attempt < maxRetries-1 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	log.Status = models.StatusFailed
	log.ErrorMsg = lastErr.Error()
	s.DB.Create(&log)

	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func (s *SMTPService) sendEmailWithConfig(config models.SMTPConfig, req EmailRequest) error {
	from := config.From
	to := []string{req.RecipientEmail}
	
	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = req.RecipientEmail
	headers["Subject"] = req.Subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"
	
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + req.Body

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	if config.UseTLS {
		return s.sendMailTLS(addr, auth, from, to, []byte(message))
	}

	return smtp.SendMail(addr, auth, from, to, []byte(message))
}

func (s *SMTPService) sendMailTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	host := strings.Split(addr, ":")[0]
	
	tlsConfig := &tls.Config{
		ServerName: host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}

	if err = client.Mail(from); err != nil {
		return err
	}

	for _, addr := range to {
		if err = client.Rcpt(addr); err != nil {
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
