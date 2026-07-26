package model

import (
	"time"

	"github.com/google/uuid"
)

type MessageStatus int

type Outbox struct {
	ID        uuid.UUID     `gorm:"message_id"`
	Payload   []byte        `gorm:"payload"`
	Status    MessageStatus `gorm:"status"`
	Timestamp time.Time     `gorm:"timestamp"`
}
