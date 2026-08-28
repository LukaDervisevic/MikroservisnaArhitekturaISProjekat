package domainevent

import "fmt"

type EventAgendaChanged struct {
	BaseEvent
	OldAgenda string `json:"oldAgenda"`
	NewAgenda string `json:"newAgenda"`
}

func (e *EventAgendaChanged) EventType() string { return TypeEventAgendaChanged }

func (e *EventAgendaChanged) Describe() string {
	return fmt.Sprintf("agenda changed from %q to %q", e.OldAgenda, e.NewAgenda)
}
