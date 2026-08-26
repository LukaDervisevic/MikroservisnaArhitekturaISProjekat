package domainevent

import "fmt"

type EventPriceChanged struct {
	BaseEvent
	OldPrice float64 `json:"oldPrice"`
	NewPrice float64 `json:"newPrice"`
}

func (e *EventPriceChanged) EventType() string { return TypeEventPriceChanged }

func (e *EventPriceChanged) Describe() string {
	return fmt.Sprintf("price changed from %.2f to %.2f", e.OldPrice, e.NewPrice)
}
