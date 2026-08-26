package aggregate

import (
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/domainevent"
	"github.com/google/uuid"
)

type EventAggregate struct {
	id              uuid.UUID
	version         int64
	exists          bool
	name            string
	cotisationPrice float64
	agenda          string
	eventType       string
	dateTime        int64
	locationID      int64
	cancelled       bool

	uncommitted []domainevent.DomainEvent
}

func NewEventAggregate(id uuid.UUID) *EventAggregate {
	return &EventAggregate{id: id}
}

func (a *EventAggregate) ID() uuid.UUID            { return a.id }
func (a *EventAggregate) Version() int64           { return a.version }
func (a *EventAggregate) Exists() bool             { return a.exists }
func (a *EventAggregate) Name() string             { return a.name }
func (a *EventAggregate) CotisationPrice() float64 { return a.cotisationPrice }
func (a *EventAggregate) Agenda() string           { return a.agenda }
func (a *EventAggregate) Type() string             { return a.eventType }
func (a *EventAggregate) DateTime() int64          { return a.dateTime }
func (a *EventAggregate) LocationID() int64        { return a.locationID }
func (a *EventAggregate) Cancelled() bool          { return a.cancelled }

func (a *EventAggregate) UncommittedEvents() []domainevent.DomainEvent {
	return a.uncommitted
}

func (a *EventAggregate) MarkCommitted() {
	a.uncommitted = nil
}

func (a *EventAggregate) Create(name string, cotisationPrice float64, agenda, eventType string, dateTime, locationID int64) error {
	if a.exists {
		return ErrAlreadyExists
	}
	if name == "" {
		return ErrNameRequired
	}
	if agenda == "" {
		return ErrAgendaRequired
	}
	if eventType == "" {
		return ErrTypeRequired
	}
	if dateTime <= 0 {
		return ErrInvalidDateTime
	}
	if locationID <= 0 {
		return ErrInvalidLocation
	}
	if cotisationPrice < 0 {
		return ErrInvalidPrice
	}

	a.raise(&domainevent.EventCreated{
		BaseEvent:       domainevent.NewBase(a.id, a.version+1),
		Name:            name,
		CotisationPrice: cotisationPrice,
		Agenda:          agenda,
		Type:            eventType,
		DateTime:        dateTime,
		LocationID:      locationID,
	})
	return nil
}

func (a *EventAggregate) Rename(newName string) error {
	if err := a.mustBeMutable(); err != nil {
		return err
	}
	if newName == "" {
		return ErrNameRequired
	}
	if newName == a.name {
		return ErrNoOpChange
	}

	a.raise(&domainevent.EventRenamed{
		BaseEvent: domainevent.NewBase(a.id, a.version+1),
		OldName:   a.name,
		NewName:   newName,
	})
	return nil
}

func (a *EventAggregate) Reschedule(newDateTime int64) error {
	if err := a.mustBeMutable(); err != nil {
		return err
	}
	if newDateTime <= 0 {
		return ErrInvalidDateTime
	}
	if newDateTime == a.dateTime {
		return ErrNoOpChange
	}

	a.raise(&domainevent.EventRescheduled{
		BaseEvent:   domainevent.NewBase(a.id, a.version+1),
		OldDateTime: a.dateTime,
		NewDateTime: newDateTime,
	})
	return nil
}

func (a *EventAggregate) Relocate(newLocationID int64) error {
	if err := a.mustBeMutable(); err != nil {
		return err
	}
	if newLocationID <= 0 {
		return ErrInvalidLocation
	}
	if newLocationID == a.locationID {
		return ErrNoOpChange
	}

	a.raise(&domainevent.EventRelocated{
		BaseEvent:     domainevent.NewBase(a.id, a.version+1),
		OldLocationID: a.locationID,
		NewLocationID: newLocationID,
	})
	return nil
}

func (a *EventAggregate) ChangePrice(newPrice float64) error {
	if err := a.mustBeMutable(); err != nil {
		return err
	}
	if newPrice < 0 {
		return ErrInvalidPrice
	}
	if newPrice == a.cotisationPrice {
		return ErrNoOpChange
	}

	a.raise(&domainevent.EventPriceChanged{
		BaseEvent: domainevent.NewBase(a.id, a.version+1),
		OldPrice:  a.cotisationPrice,
		NewPrice:  newPrice,
	})
	return nil
}

func (a *EventAggregate) Cancel(reason string) error {
	if !a.exists {
		return ErrDoesNotExist
	}
	if a.cancelled {
		return ErrAlreadyCancelled
	}

	a.raise(&domainevent.EventCancelled{
		BaseEvent: domainevent.NewBase(a.id, a.version+1),
		Reason:    reason,
	})
	return nil
}

func (a *EventAggregate) mustBeMutable() error {
	if !a.exists {
		return ErrDoesNotExist
	}
	if a.cancelled {
		return ErrEventCancelled
	}
	return nil
}

func (a *EventAggregate) raise(e domainevent.DomainEvent) {
	a.apply(e)
	a.uncommitted = append(a.uncommitted, e)
}

func (a *EventAggregate) apply(e domainevent.DomainEvent) {
	switch ev := e.(type) {
	case *domainevent.EventCreated:
		a.exists = true
		a.name = ev.Name
		a.cotisationPrice = ev.CotisationPrice
		a.agenda = ev.Agenda
		a.eventType = ev.Type
		a.dateTime = ev.DateTime
		a.locationID = ev.LocationID
	case *domainevent.EventRenamed:
		a.name = ev.NewName
	case *domainevent.EventRescheduled:
		a.dateTime = ev.NewDateTime
	case *domainevent.EventRelocated:
		a.locationID = ev.NewLocationID
	case *domainevent.EventPriceChanged:
		a.cotisationPrice = ev.NewPrice
	case *domainevent.EventCancelled:
		a.cancelled = true
	}
	a.version = e.GetVersion()
}

func (a *EventAggregate) ReplayHistory(events []domainevent.DomainEvent) {
	for _, e := range events {
		a.apply(e)
	}
}

// EventAggregateState is a snapshot of EventAggregate
type EventAggregateState struct {
	ID              uuid.UUID `json:"id"`
	Version         int64     `json:"version"`
	Exists          bool      `json:"exists"`
	Name            string    `json:"name"`
	CotisationPrice float64   `json:"cotisationPrice"`
	Agenda          string    `json:"agenda"`
	Type            string    `json:"type"`
	DateTime        int64     `json:"dateTime"`
	LocationID      int64     `json:"locationId"`
	Cancelled       bool      `json:"cancelled"`
}

func (a *EventAggregate) ToState() EventAggregateState {
	return EventAggregateState{
		ID:              a.id,
		Version:         a.version,
		Exists:          a.exists,
		Name:            a.name,
		CotisationPrice: a.cotisationPrice,
		Agenda:          a.agenda,
		Type:            a.eventType,
		DateTime:        a.dateTime,
		LocationID:      a.locationID,
		Cancelled:       a.cancelled,
	}
}

func FromState(s EventAggregateState) *EventAggregate {
	return &EventAggregate{
		id:              s.ID,
		version:         s.Version,
		exists:          s.Exists,
		name:            s.Name,
		cotisationPrice: s.CotisationPrice,
		agenda:          s.Agenda,
		eventType:       s.Type,
		dateTime:        s.DateTime,
		locationID:      s.LocationID,
		cancelled:       s.Cancelled,
	}
}
