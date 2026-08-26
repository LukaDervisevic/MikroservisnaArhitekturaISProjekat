package domainevent

import (
	"time"

	"github.com/google/uuid"
)

type DomainEvent interface {
	GetEventID() uuid.UUID
	GetAggregateID() uuid.UUID
	GetVersion() int64
	GetOccurredAt() time.Time
	EventType() string
	Describe() string
}

type BaseEvent struct {
	EventID     uuid.UUID `json:"eventId"`
	AggregateID uuid.UUID `json:"aggregateId"`
	Version     int64     `json:"version"`
	OccurredAt  time.Time `json:"occurredAt"`
}

func (b BaseEvent) GetEventID() uuid.UUID     { return b.EventID }
func (b BaseEvent) GetAggregateID() uuid.UUID { return b.AggregateID }
func (b BaseEvent) GetVersion() int64         { return b.Version }
func (b BaseEvent) GetOccurredAt() time.Time  { return b.OccurredAt }

func NewBase(aggregateID uuid.UUID, version int64) BaseEvent {
	return BaseEvent{
		EventID:     uuid.New(),
		AggregateID: aggregateID,
		Version:     version,
		OccurredAt:  time.Now().UTC(),
	}
}

const (
	TypeEventCreated      = "EventCreated"
	TypeEventRenamed      = "EventRenamed"
	TypeEventRescheduled  = "EventRescheduled"
	TypeEventRelocated    = "EventRelocated"
	TypeEventPriceChanged = "EventPriceChanged"
	TypeEventCancelled    = "EventCancelled"
)
