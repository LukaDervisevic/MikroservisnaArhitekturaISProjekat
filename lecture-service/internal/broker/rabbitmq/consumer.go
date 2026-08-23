package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Azure/go-amqp"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/mapper"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/repo"
	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/service/saga"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type Message struct {
	IdempotentKey uuid.UUID       `json:"idempotentKey"`
	SagaID        uuid.UUID       `json:"sagaId"`
	Method        string          `json:"method"`
	Body          json.RawMessage `json:"body"`
	TimeStamp     time.Time
	Retries       int
}

type ProcessedMessage struct {
	IdempotentKey uuid.UUID `gorm:"type:uuid;primaryKey"`
	Method        string
	ProcessedAt   time.Time
}

type DeadLetter struct {
	Message   Message
	Error     error
	CreatedAt time.Time
}

// sagaTimeout bounds the wait for lecture-query-service. It stays below the
// initiator's timeout so a stall surfaces here rather than at every hop at once.
const sagaTimeout = 10 * time.Second

type ConsumerConn struct {
	seen          map[uuid.UUID]struct{}
	mu            sync.RWMutex
	Environment   *rmq.Environment
	Connection    *rmq.AmqpConnection
	responders    map[string]rmq.Responder
	db            *gorm.DB
	eventRepo     repo.IEventWriteRepo
	locationRepo  repo.ILocationWriteRepo
	lecturerRepo  repo.ILecturerWriteRepo
	lectureRepo   *repo.LectureRepo
	sagaReplies   *saga.SagaReplyRegistry
	publisherConn *PublisherConn
}

func NewConsumerConn(
	ctx context.Context,
	brokerURI string,
	connOptions *rmq.AmqpConnOptions,
	db *gorm.DB,
	eventRepo repo.IEventWriteRepo,
	locationRepo repo.ILocationWriteRepo,
	lecturerRepo repo.ILecturerWriteRepo,
	lectureRepo *repo.LectureRepo,
	sagaReplies *saga.SagaReplyRegistry,
	publisherConn *PublisherConn,
) (*ConsumerConn, error) {
	env := rmq.NewEnvironment(brokerURI, connOptions)
	conn, err := env.NewConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect to broker: %w", err)
	}

	return &ConsumerConn{
		seen:          make(map[uuid.UUID]struct{}),
		Environment:   env,
		Connection:    conn,
		responders:    make(map[string]rmq.Responder),
		db:            db,
		eventRepo:     eventRepo,
		locationRepo:  locationRepo,
		lecturerRepo:  lecturerRepo,
		lectureRepo:   lectureRepo,
		sagaReplies:   sagaReplies,
		publisherConn: publisherConn,
	}, nil
}

func (b *ConsumerConn) NewQueueResponder(ctx context.Context, queueName string) error {
	if b.Connection == nil {
		return errors.New("no broker connection")
	}

	responder, err := b.Connection.NewResponder(ctx, rmq.ResponderOptions{
		RequestQueue: queueName,
		Handler:      b.handle,
	})
	if err != nil {
		return fmt.Errorf("create responder for queue %s: %w", queueName, err)
	}

	b.responders[queueName] = responder
	return nil
}

func (b *ConsumerConn) handle(ctx context.Context, request *amqp.Message) (*amqp.Message, error) {
	if len(request.Data) == 0 {
		return nil, errors.New("empty message payload")
	}

	var msg Message
	if err := json.Unmarshal(request.Data[0], &msg); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}

	log.Info().Msgf("received message %s (%s)", msg.IdempotentKey, msg.Method)

	b.mu.RLock()
	_, cached := b.seen[msg.IdempotentKey]
	b.mu.RUnlock()
	if cached {
		log.Info().Msgf("message %s already processed, acking", msg.IdempotentKey)
		return nil, nil
	}

	err := b.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Claim the key first. Unique PK violation == concurrent/duplicate delivery.
		claim := ProcessedMessage{
			IdempotentKey: msg.IdempotentKey,
			Method:        msg.Method,
			ProcessedAt:   time.Now().UTC(),
		}
		if err := tx.Create(&claim).Error; err != nil {
			return fmt.Errorf("claim idempotency key: %w", err)
		}
		return b.dispatch(ctx, tx, msg)
	})

	if isDuplicateKey(err) {
		b.remember(msg.IdempotentKey)
		return nil, nil // committed by someone else; still a success
	}

	// Reply upstream only once the local tx has resolved, so we never report a
	// commit for a transaction that then failed to commit.
	b.replyUpstream(ctx, msg, err)

	if err != nil {
		log.Error().Err(err).Msgf("tx failed for message %s", msg.IdempotentKey)
		return nil, err // rolled back, safe to redeliver
	}

	b.remember(msg.IdempotentKey)
	log.Info().Msgf("processed %s for message %s", msg.Method, msg.IdempotentKey)
	return nil, nil
}

// replyUpstream tells the previous service in the saga chain whether this
// service committed its own change.
func (b *ConsumerConn) replyUpstream(ctx context.Context, msg Message, txErr error) {
	var replyQueue, replyMethod string
	switch msg.Method {
	case "UpdateLecturerSAGA":
		replyQueue, replyMethod = os.Getenv("RABBITMQ_REPLY_TO_LECTURER_QUEUE"), "UpdateLecturerSAGAReply"
	default:
		return
	}

	reply := model.CommitReply()
	if txErr != nil {
		reply = model.RollbackReply(txErr)
	}

	if err := b.publisherConn.PublishSaga(ctx, replyQueue, msg.SagaID, replyMethod, reply); err != nil {
		log.Error().Err(err).Msgf("failed to reply to saga %s on queue %s", msg.SagaID, replyQueue)
	}
}

func (b *ConsumerConn) dispatch(ctx context.Context, tx *gorm.DB, msg Message) error {
	events := b.eventRepo.WithTx(tx)
	locations := b.locationRepo.WithTx(tx)
	lecturers := b.lecturerRepo.WithTx(tx)

	switch msg.Method {
	case "CreateEvent":
		event, err := decode[model.Event](msg.Body)
		if err != nil {
			return err
		}
		return events.CreateEvent(ctx, event)

	case "UpdateEvent":
		event, err := decode[model.Event](msg.Body)
		if err != nil {
			return err
		}
		return events.UpdateEvent(ctx, event)

	case "DeleteEvent":
		event, err := decode[model.Event](msg.Body)
		if err != nil {
			return err
		}
		return events.DeleteEvent(ctx, event.Id)

	case "CreateLocation":
		location, err := decode[model.Location](msg.Body)
		if err != nil {
			return err
		}
		return locations.CreateLocation(ctx, location)

	case "UpdateLocation":
		location, err := decode[model.Location](msg.Body)
		if err != nil {
			return err
		}
		return locations.UpdateLocation(ctx, location)

	case "DeleteLocation":
		location, err := decode[model.Location](msg.Body)
		if err != nil {
			return err
		}
		return locations.DeleteLocation(ctx, location.Id)

	case "CreateLecturer":
		lecturer, err := decode[model.Lecturer](msg.Body)
		if err != nil {
			return err
		}
		return lecturers.CreateLecturer(ctx, lecturer)

	case "UpdateLecturer":
		lecturer, err := decode[model.Lecturer](msg.Body)
		if err != nil {
			return err
		}
		return lecturers.UpdateLecturer(ctx, lecturer)

	case "DeleteLecturer":
		lecturer, err := decode[model.Lecturer](msg.Body)
		if err != nil {
			return err
		}
		return lecturers.DeleteLecturer(ctx, lecturer.Id)

	// Middle of the saga chain: apply the change locally, then hand the saga to
	// lecture-query-service. Returning an error here rolls this tx back, and
	// handle() turns that into a rollback reply for lecturer-service.
	case "UpdateLecturerSAGA":
		lecturer, err := decode[model.Lecturer](msg.Body)
		if err != nil {
			return err
		}
		if err := lecturers.UpdateLecturer(ctx, lecturer); err != nil {
			return fmt.Errorf("update lecturer replica: %w", err)
		}

		// Rebuild the read-model rows this lecturer appears on. Reading through
		// tx is what makes the projections carry the update applied just above.
		lectures, err := b.lectureRepo.WithTx(tx).ListAllLecturesByLecturerID(ctx, lecturer.Id)
		if err != nil {
			return fmt.Errorf("list lectures for lecturer %d: %w", lecturer.Id, err)
		}

		projections := make([]*model.LectureQuery, 0, len(lectures))
		for i := range lectures {
			projection := mapper.MapLectureToQuery(&lectures[i])
			if projection == nil {
				return fmt.Errorf("lecture %d is missing event or lecturer data", lectures[i].LectureID)
			}
			projections = append(projections, projection)
		}

		return b.awaitDownstream(ctx, msg.SagaID,
			os.Getenv("RABBITMQ_LECTURE_TO_LECTURE_QUERY_QUEUE"),
			"UpdateLectureQuerySAGA", projections)

	case "UpdateLectureQuerySAGAReply":
		reply, err := decode[model.SagaReply](msg.Body)
		if err != nil {
			return err
		}
		b.sagaReplies.Resolve(msg.SagaID, reply.Err())
		return nil

	default:
		return fmt.Errorf("unknown method: %s", msg.Method)
	}
}

// awaitDownstream forwards the saga to the next service and blocks until that
// service reports back. The reply arrives on a different queue served by its own
// responder goroutine, so this wait cannot deadlock the reply path.
func (b *ConsumerConn) awaitDownstream(ctx context.Context, sagaID uuid.UUID, queue, method string, body any) error {
	ch := b.sagaReplies.Register(sagaID)
	defer b.sagaReplies.Unregister(sagaID)

	if err := b.publisherConn.PublishSaga(ctx, queue, sagaID, method, body); err != nil {
		return fmt.Errorf("dispatch %s: %w", method, err)
	}

	select {
	case err := <-ch:
		return err
	case <-time.After(sagaTimeout):
		return fmt.Errorf("timed out waiting for %s reply", method)
	}
}

func decode[T any](raw json.RawMessage) (*T, error) {
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode body as %T: %w", out, err)
	}
	return &out, nil
}

func (b *ConsumerConn) remember(key uuid.UUID) {
	b.mu.Lock()
	b.seen[key] = struct{}{}
	b.mu.Unlock()
}

const pgUniqueViolation = "23505"

func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}

	// Works when gorm.Config{TranslateError: true} is set.
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == pgUniqueViolation
	}

	return false
}
