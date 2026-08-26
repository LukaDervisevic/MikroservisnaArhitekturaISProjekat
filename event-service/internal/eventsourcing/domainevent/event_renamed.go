package domainevent

import "fmt"

type EventRenamed struct {
	BaseEvent
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}

func (e *EventRenamed) EventType() string { return TypeEventRenamed }

func (e *EventRenamed) Describe() string {
	return fmt.Sprintf("renamed from %q to %q", e.OldName, e.NewName)
}
