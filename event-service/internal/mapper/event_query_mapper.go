package mapper

import "github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"

func MapEventToQuery(event *model.Event, location *model.Location) *model.EventWithLocation {

	return &model.EventWithLocation{
		EventId:              event.Id,
		EventCotisationPrice: event.CotisationPrice,
		EventAgenda:          event.Agenda,
		EventDateTime:        event.DateTime,
		EventName:            event.Name,
		EventType:            event.Type,
		LocationID:           location.Id,
		LocationName:         location.Name,
		LocationAddress:      location.Address,
		LocationCapacity:     location.Capacity,
	}
}
