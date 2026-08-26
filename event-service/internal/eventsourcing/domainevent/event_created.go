package domainevent

import "fmt"

type EventCreated struct {
	BaseEvent
	Name            string  `json:"name"`
	CotisationPrice float64 `json:"cotisationPrice"`
	Agenda          string  `json:"agenda"`
	Type            string  `json:"type"`
	DateTime        int64   `json:"dateTime"`
	LocationID      int64   `json:"locationId"`
}

func (e *EventCreated) EventType() string { return TypeEventCreated }

func (e *EventCreated) Describe() string {
	return fmt.Sprintf("created %q (type=%s, price=%.2f, location=%d)", e.Name, e.Type, e.CotisationPrice, e.LocationID)
}
