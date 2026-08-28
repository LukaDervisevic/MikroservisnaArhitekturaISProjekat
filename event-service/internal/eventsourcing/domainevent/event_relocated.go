package domainevent

import "fmt"

type EventRelocated struct {
	BaseEvent
	OldLocationID int64 `json:"oldLocationId"`
	NewLocationID int64 `json:"newLocationId"`
}

func (e *EventRelocated) EventType() string { return TypeEventRelocated }

func (e *EventRelocated) Describe() string {
	return fmt.Sprintf("relocated from location %d to location %d", e.OldLocationID, e.NewLocationID)
}
