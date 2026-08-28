package domainevent

import "fmt"

type EventTypeChanged struct {
	BaseEvent
	OldType string `json:"oldType"`
	NewType string `json:"newType"`
}

func (e *EventTypeChanged) EventType() string { return TypeEventTypeChanged }

func (e *EventTypeChanged) Describe() string {
	return fmt.Sprintf("type changed from %q to %q", e.OldType, e.NewType)
}
