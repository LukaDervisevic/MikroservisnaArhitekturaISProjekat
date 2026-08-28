package service

import (
	"context"
	"fmt"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/aggregate"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/domainevent"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/snapshot"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/eventsourcing/store"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/event-service/internal/repo"
	"gorm.io/gorm"
)

const snapshotEvery = 5

type EventAggregateService struct {
	db         *gorm.DB
	store      store.EventStore
	snapshots  snapshot.SnapshotStore
	projection *repo.EventRepo
}

func NewEventAggregateService(
	db *gorm.DB,
	eventStore store.EventStore,
	snapshotStore snapshot.SnapshotStore,
	projection *repo.EventRepo,
) *EventAggregateService {
	return &EventAggregateService{db: db, store: eventStore, snapshots: snapshotStore, projection: projection}
}

func (s *EventAggregateService) DB() *gorm.DB { return s.db }

func (s *EventAggregateService) NextAggregateID(ctx context.Context) (int64, error) {
	return s.store.NextAggregateID(ctx)
}

func (s *EventAggregateService) Load(ctx context.Context, id int64) (*aggregate.EventAggregate, error) {
	return s.load(ctx, s.store, s.snapshots, id)
}

func (s *EventAggregateService) LoadTx(ctx context.Context, tx *gorm.DB, id int64) (*aggregate.EventAggregate, error) {
	return s.load(ctx, s.store.WithTx(tx), s.snapshots.WithTx(tx), id)
}

func (s *EventAggregateService) load(
	ctx context.Context,
	eventStore store.EventStore,
	snapshots snapshot.SnapshotStore,
	id int64,
) (*aggregate.EventAggregate, error) {
	snap, err := snapshots.GetLatest(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("load snapshot: %w", err)
	}

	var agg *aggregate.EventAggregate
	fromVersion := int64(1)
	if snap != nil {
		agg = aggregate.FromState(*snap)
		fromVersion = snap.Version + 1
	} else {
		agg = aggregate.NewEventAggregate(id)
	}

	tail, err := eventStore.LoadFrom(ctx, id, fromVersion)
	if err != nil {
		return nil, fmt.Errorf("load event tail: %w", err)
	}
	agg.ReplayHistory(tail)
	return agg, nil
}

func (s *EventAggregateService) AppendTx(ctx context.Context, tx *gorm.DB, agg *aggregate.EventAggregate) error {
	uncommitted := agg.UncommittedEvents()
	if len(uncommitted) == 0 {
		return nil
	}
	fromVersion := agg.Version() - int64(len(uncommitted))
	if err := s.store.WithTx(tx).Append(ctx, uncommitted); err != nil {
		return err
	}
	agg.MarkCommitted()

	if err := s.syncProjection(ctx, tx, agg.ToState()); err != nil {
		return fmt.Errorf("sync event projection %d: %w", agg.ID(), err)
	}

	if agg.Version()/snapshotEvery > fromVersion/snapshotEvery {
		if err := s.snapshots.WithTx(tx).Save(ctx, agg.ToState()); err != nil {
			return fmt.Errorf("auto snapshot at version %d: %w", agg.Version(), err)
		}
	}
	return nil
}

func (s *EventAggregateService) syncProjection(ctx context.Context, tx *gorm.DB, st aggregate.EventAggregateState) error {
	proj := s.projection.WithTx(tx)
	if st.Cancelled || !st.Exists {
		return proj.DeleteEvent(ctx, st.ID)
	}
	return proj.UpsertEvent(ctx, &model.Event{
		Id:              st.ID,
		Name:            st.Name,
		CotisationPrice: st.CotisationPrice,
		Agenda:          st.Agenda,
		Type:            st.Type,
		DateTime:        st.DateTime,
		LocationID:      st.LocationID,
	})
}

func (s *EventAggregateService) DeleteAggregateTx(ctx context.Context, tx *gorm.DB, id int64) error {
	if err := s.store.WithTx(tx).Delete(ctx, id); err != nil {
		return fmt.Errorf("delete event log %d: %w", id, err)
	}
	if err := tx.WithContext(ctx).
		Where("aggregate_id = ?", id).
		Delete(&snapshot.EventAggregateSnapshot{}).Error; err != nil {
		return fmt.Errorf("delete event snapshots %d: %w", id, err)
	}
	if err := s.projection.WithTx(tx).DeleteEvent(ctx, id); err != nil {
		return fmt.Errorf("delete event projection %d: %w", id, err)
	}
	return nil
}

func (s *EventAggregateService) GetHistory(ctx context.Context, id int64) ([]domainevent.DomainEvent, error) {
	return s.store.Load(ctx, id)
}

func (s *EventAggregateService) CreateSnapshot(ctx context.Context, agg *aggregate.EventAggregate) error {
	return s.snapshots.Save(ctx, agg.ToState())
}
