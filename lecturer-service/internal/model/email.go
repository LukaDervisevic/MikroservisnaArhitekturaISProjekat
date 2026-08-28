package model

import (
	"time"

	"github.com/google/uuid"
)

type EmailMessage struct {
	IdempotentKey uuid.UUID `json:"idempotent_key"`
	To            string    `json:"to"`
	Subject       string    `json:"subject"`
	Body          string    `json:"body"`
	EnqueuedAt    time.Time `json:"enqueued_at"`
	RetryCount    int       `json:"retry_count"`
	ForceFail     bool      `json:"force_fail"`
}
