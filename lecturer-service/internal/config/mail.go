package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type TLSMode string

const (
	TLSStartTLS TLSMode = "starttls"
	TLSImplicit TLSMode = "tls"
	TLSNone     TLSMode = "none"
)

type SMTPConfig struct {
	Host        string
	Port        string
	Username    string
	Password    string
	TLSMode     TLSMode
	FromAddress string
	FromName    string
	Timeout     time.Duration
}

func (c SMTPConfig) Addr() string { return c.Host + ":" + c.Port }

type MailConfig struct {
	Queue      string
	DLQQueue   string
	RateLimit  int
	RatePeriod time.Duration
	MaxRetries int
	SMTP       SMTPConfig
}

const (
	defaultMailQueue    = "mail.send"
	defaultMailDLQQueue = "mail.dlq"
	defaultRateLimit    = 10
	defaultRatePeriod   = time.Minute
	defaultMaxRetries   = 10

	defaultSMTPHost    = "smtp.gmail.com"
	defaultSMTPPort    = "587"
	defaultFromName    = "MAIS Lecturer Service"
	defaultSendTimeout = 30 * time.Second
)

func LoadMailConfig() MailConfig {
	return MailConfig{
		Queue:      envString("RABBITMQ_MAIL_QUEUE", defaultMailQueue),
		DLQQueue:   envString("RABBITMQ_MAIL_DLQ_QUEUE", defaultMailDLQQueue),
		RateLimit:  envInt("MAIL_RATE_LIMIT", defaultRateLimit),
		RatePeriod: envDuration("MAIL_RATE_PERIOD", defaultRatePeriod),
		MaxRetries: envInt("MAIL_MAX_RETRIES", defaultMaxRetries),
		SMTP: SMTPConfig{
			Host:        envString("MAIL_SMTP_HOST", defaultSMTPHost),
			Port:        envString("MAIL_SMTP_PORT", defaultSMTPPort),
			Username:    os.Getenv("MAIL_SMTP_USERNAME"),
			Password:    os.Getenv("MAIL_SMTP_PASSWORD"),
			TLSMode:     envTLSMode("MAIL_SMTP_TLS"),
			FromAddress: envString("MAIL_FROM_ADDRESS", os.Getenv("MAIL_SMTP_USERNAME")),
			FromName:    envString("MAIL_FROM_NAME", defaultFromName),
			Timeout:     envDuration("MAIL_SEND_TIMEOUT", defaultSendTimeout),
		},
	}
}

func envTLSMode(key string) TLSMode {
	switch strings.ToLower(os.Getenv(key)) {
	case string(TLSStartTLS):
		return TLSStartTLS
	case string(TLSImplicit):
		return TLSImplicit
	case string(TLSNone):
		return TLSNone
	default:
		if os.Getenv("MAIL_SMTP_USERNAME") != "" {
			return TLSStartTLS
		}
		return TLSNone
	}
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
