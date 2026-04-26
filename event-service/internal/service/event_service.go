package service

import (
	"context"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type EventManagementService struct {
	eventRepo *repo.EventRepo
	db        *gorm.DB
}

func NewEventManagementService(
	db *gorm.DB,
	eventRepo *repo.EventRepo,
) *EventManagementService {
	return &EventManagementService{
		eventRepo: eventRepo,
		db:        db,
	}
}

type CreateLocationInput struct {
	Name     string
	Address  string
	Capacity int64
}

type GetLocationByIDInput struct {
	Id int64
}

type GetLocationByNameInput struct {
	Name string
}

type ListLocationsInput struct {
	Page        int
	PageSize    int
	MinCapacity int64
	MaxCapacity int64
}

type UpdateLocationInput struct {
	Id       int64
	Name     string
	Address  string
	Capacity int64
}

type DeleteLocationInput struct {
	Id int64
}

type CreateEventInput struct {
	Name            string
	CotisationPrice float64
	Agenda          string
	Type            string
	DateTime        int64
	LocationID      int64
}

type GetEventByIDInput struct {
	Id int64
}

type GetEventByNameInput struct {
	Name string
}

type ListEventsInput struct {
	Page       int
	PageSize   int
	Type       string
	FromDate   int64
	ToDate     int64
	LocationID int64
}

type UpdateEventInput struct {
	Id              int64
	Name            string
	CotisationPrice float64
	Agenda          string
	Type            string
	DateTime        int64
	LocationID      int64
}

type DeleteEventInput struct {
	Id int64
}

type CreateLectureInput struct {
	EventID    int64
	LecturerID int64
	Name       string
	Duration   int64
}

type GetLectureByIDInput struct {
	Id int64
}

type GetLectureByNameInput struct {
	Name string
}

type ListLecturesByEventIDInput struct {
	EventID  int64
	Page     int
	PageSize int
}

type ListLecturesByLecturerIDInput struct {
	LecturerID int64
	Page       int
	PageSize   int
}

type UpdateLectureInput struct {
	LectureID  int64
	EventID    int64
	LecturerID int64
	Name       string
	Duration   int64
}

type DeleteLectureInput struct {
	LectureID int64
}

func (s *EventManagementService) CreateLocation(ctx context.Context, input *CreateLocationInput) (*model.Location, error) {
	location := &model.Location{
		Name:     input.Name,
		Address:  input.Address,
		Capacity: input.Capacity,
	}
	if err := s.eventRepo.CreateLocation(ctx, location); err != nil {
		return nil, status.Error(codes.Internal, "failed to create location")
	}
	return location, nil
}

func (s *EventManagementService) GetLocationByID(ctx context.Context, input *GetLocationByIDInput) (*model.Location, error) {
	location, err := s.eventRepo.GetLocationByID(ctx, input.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve location")
	}
	if location == nil {
		return nil, status.Error(codes.NotFound, "location not found")
	}
	return location, nil
}

func (s *EventManagementService) GetLocationByName(ctx context.Context, input *GetLocationByNameInput) (*model.Location, error) {
	location, err := s.eventRepo.GetLocationByName(ctx, input.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve location")
	}
	if location == nil {
		return nil, status.Error(codes.NotFound, "location not found")
	}
	return location, nil
}

func (s *EventManagementService) ListLocations(ctx context.Context, input *ListLocationsInput) ([]model.Location, int64, error) {
	locations, totalCount, err := s.eventRepo.ListLocations(ctx, repo.ListLocationsFilter{
		Page:        input.Page,
		PageSize:    input.PageSize,
		MinCapacity: input.MinCapacity,
		MaxCapacity: input.MaxCapacity,
	})
	if err != nil {
		return nil, 0, status.Error(codes.Internal, "failed to list locations")
	}
	return locations, totalCount, nil
}

func (s *EventManagementService) UpdateLocation(ctx context.Context, input *UpdateLocationInput) error {
	location, err := s.eventRepo.GetLocationByID(ctx, input.Id)
	if err != nil {
		return status.Error(codes.Internal, "failed to retrieve location")
	}
	if location == nil {
		return status.Error(codes.NotFound, "location not found")
	}
	location.Name = input.Name
	location.Address = input.Address
	location.Capacity = input.Capacity
	if err := s.eventRepo.UpdateLocation(ctx, location); err != nil {
		return status.Error(codes.Internal, "failed to update location")
	}
	return nil
}

func (s *EventManagementService) DeleteLocation(ctx context.Context, input *DeleteLocationInput) (*model.Location, error) {
	location, err := s.eventRepo.GetLocationByID(ctx, input.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve location")
	}
	if location == nil {
		return nil, status.Error(codes.NotFound, "location not found")
	}
	if err := s.eventRepo.DeleteLocation(ctx, input.Id); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete location")
	}
	return location, nil
}

func (s *EventManagementService) CreateEvent(ctx context.Context, input *CreateEventInput) (*model.Event, error) {
	location, err := s.eventRepo.GetLocationByID(ctx, input.LocationID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to verify location")
	}
	if location == nil {
		return nil, status.Error(codes.NotFound, "location not found")
	}
	event := &model.Event{
		Name:            input.Name,
		CotisationPrice: input.CotisationPrice,
		Agenda:          input.Agenda,
		Type:            input.Type,
		DateTime:        input.DateTime,
		LocationID:      input.LocationID,
	}
	if err := s.eventRepo.CreateEvent(ctx, event); err != nil {
		return nil, status.Error(codes.Internal, "failed to create event")
	}
	return event, nil
}

func (s *EventManagementService) GetEventByID(ctx context.Context, input *GetEventByIDInput) (*model.Event, error) {
	event, err := s.eventRepo.GetEventByID(ctx, input.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve event")
	}
	if event == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}
	return event, nil
}

func (s *EventManagementService) GetEventByName(ctx context.Context, input *GetEventByNameInput) (*model.Event, error) {
	event, err := s.eventRepo.GetEventByName(ctx, input.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve event")
	}
	if event == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}
	return event, nil
}

func (s *EventManagementService) ListEvents(ctx context.Context, input *ListEventsInput) ([]model.Event, int64, error) {
	events, totalCount, err := s.eventRepo.ListEvents(ctx, repo.ListEventsFilter{
		Page:       input.Page,
		PageSize:   input.PageSize,
		Type:       input.Type,
		FromDate:   input.FromDate,
		ToDate:     input.ToDate,
		LocationID: input.LocationID,
	})
	if err != nil {
		return nil, 0, status.Error(codes.Internal, "failed to list events")
	}
	return events, totalCount, nil
}

func (s *EventManagementService) UpdateEvent(ctx context.Context, input *UpdateEventInput) (*model.Event, error) {
	event, err := s.eventRepo.GetEventByID(ctx, input.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve event")
	}
	if event == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}
	if input.LocationID != 0 && input.LocationID != event.LocationID {
		location, err := s.eventRepo.GetLocationByID(ctx, input.LocationID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to verify location")
		}
		if location == nil {
			return nil, status.Error(codes.NotFound, "location not found")
		}
		event.LocationID = input.LocationID
	}
	event.Name = input.Name
	event.CotisationPrice = input.CotisationPrice
	event.Agenda = input.Agenda
	event.Type = input.Type
	event.DateTime = input.DateTime
	if err := s.eventRepo.UpdateEvent(ctx, event); err != nil {
		return nil, status.Error(codes.Internal, "failed to update event")
	}
	return event, nil
}

func (s *EventManagementService) DeleteEvent(ctx context.Context, input *DeleteEventInput) (*model.Event, error) {
	event, err := s.eventRepo.GetEventByID(ctx, input.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve event")
	}
	if event == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}
	if err := s.eventRepo.DeleteEvent(ctx, input.Id); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete event")
	}
	return event, nil
}

func (s *EventManagementService) CreateLecture(ctx context.Context, input *CreateLectureInput) (*model.Lecture, error) {
	event, err := s.eventRepo.GetEventByID(ctx, input.EventID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to verify event")
	}
	if event == nil {
		return nil, status.Error(codes.NotFound, "event not found")
	}
	lecture := &model.Lecture{
		EventID:    input.EventID,
		LecturerID: input.LecturerID,
		Name:       input.Name,
		Duration:   input.Duration,
	}
	if err := s.eventRepo.CreateLecture(ctx, lecture); err != nil {
		return nil, status.Error(codes.Internal, "failed to create lecture")
	}
	return lecture, nil
}

func (s *EventManagementService) GetLectureByID(ctx context.Context, input *GetLectureByIDInput) (*model.Lecture, error) {
	lecture, err := s.eventRepo.GetLectureByID(ctx, input.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecture")
	}
	if lecture == nil {
		return nil, status.Error(codes.NotFound, "lecture not found")
	}
	return lecture, nil
}

func (s *EventManagementService) GetLectureByName(ctx context.Context, input *GetLectureByNameInput) (*model.Lecture, error) {
	lecture, err := s.eventRepo.GetLectureByName(ctx, input.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecture")
	}
	if lecture == nil {
		return nil, status.Error(codes.NotFound, "lecture not found")
	}
	return lecture, nil
}

func (s *EventManagementService) ListLecturesByEventID(ctx context.Context, input *ListLecturesByEventIDInput) ([]model.Lecture, int64, error) {
	lectures, totalCount, err := s.eventRepo.ListLecturesByEventID(ctx, repo.ListLecturesByEventIDFilter{
		EventID:  input.EventID,
		Page:     input.Page,
		PageSize: input.PageSize,
	})
	if err != nil {
		return nil, 0, status.Error(codes.Internal, "failed to list lectures")
	}
	return lectures, totalCount, nil
}

func (s *EventManagementService) ListLecturesByLecturerID(ctx context.Context, input *ListLecturesByLecturerIDInput) ([]model.Lecture, int64, error) {
	lectures, totalCount, err := s.eventRepo.ListLecturesByLecturerID(ctx, repo.ListLecturesByLecturerIDFilter{
		LecturerID: input.LecturerID,
		Page:       input.Page,
		PageSize:   input.PageSize,
	})
	if err != nil {
		return nil, 0, status.Error(codes.Internal, "failed to list lectures")
	}
	return lectures, totalCount, nil
}

func (s *EventManagementService) UpdateLecture(ctx context.Context, input *UpdateLectureInput) (*model.Lecture, error) {
	lecture, err := s.eventRepo.GetLectureByID(ctx, input.LectureID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecture")
	}
	if lecture == nil {
		return nil, status.Error(codes.NotFound, "lecture not found")
	}
	if input.EventID != 0 && input.EventID != lecture.EventID {
		event, err := s.eventRepo.GetEventByID(ctx, input.EventID)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to verify event")
		}
		if event == nil {
			return nil, status.Error(codes.NotFound, "event not found")
		}
		lecture.EventID = input.EventID
	}
	lecture.LecturerID = input.LecturerID
	lecture.Name = input.Name
	lecture.Duration = input.Duration
	if err := s.eventRepo.UpdateLecture(ctx, lecture); err != nil {
		return nil, status.Error(codes.Internal, "failed to update lecture")
	}
	return lecture, nil
}

func (s *EventManagementService) DeleteLecture(ctx context.Context, input *DeleteLectureInput) (*model.Lecture, error) {
	lecture, err := s.eventRepo.GetLectureByID(ctx, input.LectureID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve lecture")
	}
	if lecture == nil {
		return nil, status.Error(codes.NotFound, "lecture not found")
	}
	if err := s.eventRepo.DeleteLecture(ctx, input.LectureID); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete lecture")
	}
	return lecture, nil
}
