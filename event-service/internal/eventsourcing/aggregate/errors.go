package aggregate

import "errors"

var (
	ErrAlreadyExists    = errors.New("event already exists")
	ErrDoesNotExist     = errors.New("event does not exist")
	ErrEventCancelled   = errors.New("event is cancelled and can no longer be changed")
	ErrAlreadyCancelled = errors.New("event is already cancelled")
	ErrNameRequired     = errors.New("name is required")
	ErrAgendaRequired   = errors.New("agenda is required")
	ErrTypeRequired     = errors.New("type is required")
	ErrInvalidDateTime  = errors.New("date/time must be a valid future timestamp")
	ErrInvalidLocation  = errors.New("location id must be positive")
	ErrInvalidPrice     = errors.New("cotisation price cannot be negative")
	ErrNoOpChange       = errors.New("new value is the same as the current value")
)
