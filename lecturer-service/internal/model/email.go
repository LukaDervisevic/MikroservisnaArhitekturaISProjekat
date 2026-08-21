package model

import "github.com/google/uuid"

type EmailMessage struct {
	IdempotentKey uuid.UUID `json:"idempotent_key"`
	To            string    `json:"to"`
	Subject       string    `json:"subject"`
	Body          string    `json:"body"`
	RetryCount    int       `json:"retry_count"`
	ForceFail     bool      `json:"force_fail"`
}
