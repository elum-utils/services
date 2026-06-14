package callback

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	callbacksqlc "github.com/elum-utils/services/internal/utils/callback/sqlc"
)

const (
	DefaultSourceService = "service"
	JSONContentType      = "application/json"

	DefaultTable  = "clb_event"
	PaymentTable  = "payment_clb_event"
	CPATable      = "cpa_clb_event"
	PromoTable    = "promo_clb_event"
	CalendarTable = "calendar_clb_event"
	TasksTable    = "tasks_clb_event"
)

var (
	ErrNotLeased        = errors.New("callback: event is not leased by worker")
	tableNameExpression = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
)

type dbtx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Store struct {
	db        *sql.DB
	executor  dbtx
	tableName string
}

type CreateParams struct {
	SourceService      string
	EventType          string
	EventKey           string
	IdempotencyKey     string
	Payload            []byte
	PayloadContentType string
	NextAttemptAt      time.Time
}

type LeaseParams struct {
	SourceService string
	WorkerID      string
	Limit         int32
	LeaseTimeout  time.Duration
}

type FailParams struct {
	ID       uint64
	WorkerID string
	Error    string
	Attempt  uint32
	FailedAt time.Time
}

type AdminListEventsParams struct {
	SourceService string
	EventType     string
	Status        string
	Limit         int32
	Offset        int32
}

type Event struct {
	ID                 uint64
	SourceService      string
	EventType          string
	EventKey           string
	IdempotencyKey     string
	Payload            []byte
	PayloadContentType string
	Status             string
	AttemptCount       uint32
	NextAttemptAt      time.Time
	LockedBy           *string
	LockedUntil        *time.Time
	DeliveredAt        *time.Time
	RejectedAt         *time.Time
	LastError          *string
	RejectReason       *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func New(db *sql.DB) *Store {
	return NewWithTable(db, DefaultTable)
}

func NewWithTable(db *sql.DB, tableName string) *Store {
	tableName = normalizeTableName(tableName)
	return &Store{db: db, executor: db, tableName: tableName}
}

func (s *Store) WithTx(tx *sql.Tx) *Store {
	if s == nil {
		return nil
	}
	return &Store{db: s.db, executor: tx, tableName: s.tableName}
}

func (s *Store) Close() error { return nil }

func Bootstrap(ctx context.Context, db *sql.DB) error {
	return BootstrapTable(ctx, db, DefaultTable)
}

func BootstrapTable(ctx context.Context, db *sql.DB, tableName string) error {
	if db == nil {
		return errors.New("callback: nil db")
	}
	tableName = normalizeTableName(tableName)
	statement := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    source_service VARCHAR(64) NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    event_key VARCHAR(128) NOT NULL,
    idempotency_key VARCHAR(191) NOT NULL,
    payload LONGBLOB NOT NULL,
    payload_content_type VARCHAR(64) NOT NULL DEFAULT 'application/json',
    status ENUM('pending', 'processing', 'ok', 'reject') NOT NULL DEFAULT 'pending',
    attempt_count INT UNSIGNED NOT NULL DEFAULT 0,
    next_attempt_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    locked_by VARCHAR(128) NULL,
    locked_until DATETIME NULL,
    delivered_at DATETIME NULL,
    rejected_at DATETIME NULL,
    last_error TEXT NULL,
    reject_reason TEXT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY callback_event_key_uq (source_service, event_key),
    UNIQUE KEY callback_idempotency_key_uq (idempotency_key),
    KEY callback_due_idx (status, next_attempt_at, locked_until, id),
    KEY callback_type_idx (event_type, status, created_at)
)`, quoteIdentifier(tableName))
	if _, err := db.ExecContext(ctx, statement); err != nil {
		return fmt.Errorf("callback schema statement failed for %s: %w", tableName, err)
	}
	return nil
}

func (s *Store) CreateEvent(ctx context.Context, params CreateParams) (uint64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}
	sourceService := params.SourceService
	if sourceService == "" {
		sourceService = DefaultSourceService
	}
	contentType := params.PayloadContentType
	if contentType == "" {
		contentType = JSONContentType
	}
	nextAttemptAt := params.NextAttemptAt
	if nextAttemptAt.IsZero() {
		nextAttemptAt = time.Now()
	}
	idempotencyKey := strings.TrimSpace(params.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = params.EventKey
	}
	if idempotencyKey == "" {
		return 0, errors.New("callback: idempotency key is required")
	}
	result, err := s.executor.ExecContext(ctx, fmt.Sprintf(`
INSERT INTO %s (
    source_service, event_type, event_key, idempotency_key,
    payload, payload_content_type, next_attempt_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE id = LAST_INSERT_ID(id)`, s.table()),
		sourceService, params.EventType, params.EventKey, idempotencyKey,
		params.Payload, contentType, nextAttemptAt,
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return uint64(id), err
}

func (s *Store) GetEvent(ctx context.Context, id uint64) (Event, error) {
	if err := s.validate(); err != nil {
		return Event{}, err
	}
	row := s.executor.QueryRowContext(ctx, fmt.Sprintf(`
SELECT %s FROM %s WHERE id = ? LIMIT 1`, eventColumns, s.table()), id)
	value, err := scanEvent(row.Scan)
	if err != nil {
		return Event{}, err
	}
	return mapEvent(value), nil
}

func (s *Store) AdminListEvents(ctx context.Context, params AdminListEventsParams) ([]Event, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	limit, offset := normalizePage(params.Limit, params.Offset)
	rows, err := s.executor.QueryContext(ctx, fmt.Sprintf(`
SELECT %s FROM %s
WHERE (? = '' OR source_service = ?)
  AND (? = '' OR event_type = ?)
  AND (? = '' OR CAST(status AS CHAR) = ?)
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?`, eventColumns, s.table()),
		params.SourceService, params.SourceService,
		params.EventType, params.EventType,
		params.Status, params.Status,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Event, 0)
	for rows.Next() {
		value, err := scanEvent(rows.Scan)
		if err != nil {
			return nil, err
		}
		result = append(result, mapEvent(value))
	}
	return result, rows.Err()
}

func (s *Store) AdminRetryEventNow(ctx context.Context, id uint64) (int64, error) {
	return s.execRows(ctx, `
SET status = 'pending', next_attempt_at = NOW(), locked_by = NULL,
    locked_until = NULL, last_error = NULL, updated_at = NOW()
WHERE id = ? AND status IN ('pending', 'processing')`, id)
}

func (s *Store) AdminMarkEventOK(ctx context.Context, id uint64) (int64, error) {
	return s.execRows(ctx, `
SET status = 'ok', delivered_at = NOW(), locked_by = NULL,
    locked_until = NULL, last_error = NULL, updated_at = NOW()
WHERE id = ? AND status IN ('pending', 'processing')`, id)
}

func (s *Store) AdminMarkEventReject(ctx context.Context, id uint64, reason string) (int64, error) {
	return s.execRows(ctx, `
SET status = 'reject', rejected_at = NOW(), reject_reason = ?,
    locked_by = NULL, locked_until = NULL, updated_at = NOW()
WHERE id = ? AND status IN ('pending', 'processing')`, nullableString(reason), id)
}

func (s *Store) AdminResetExpiredProcessing(ctx context.Context) (int64, error) {
	return s.execRows(ctx, `
SET status = 'pending', locked_by = NULL, locked_until = NULL,
    next_attempt_at = NOW(), updated_at = NOW()
WHERE status = 'processing' AND locked_until IS NOT NULL AND locked_until <= NOW()`)
}

func (s *Store) LeaseEvents(ctx context.Context, params LeaseParams) ([]storedEvent, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	workerID := normalizeWorkerID(params.WorkerID)
	limit := params.Limit
	if limit <= 0 {
		limit = 1
	}
	leaseTimeout := params.LeaseTimeout
	if leaseTimeout <= 0 {
		leaseTimeout = time.Minute
	}

	var leased []storedEvent
	if err := s.withTx(ctx, func(txStore *Store) error {
		rows, err := txStore.executor.QueryContext(ctx, fmt.Sprintf(`
SELECT %s FROM %s
WHERE (? = '' OR source_service = ?)
  AND status IN ('pending', 'processing')
  AND next_attempt_at <= NOW()
  AND (locked_until IS NULL OR locked_until <= NOW())
ORDER BY next_attempt_at, id
LIMIT ?
FOR UPDATE SKIP LOCKED`, eventColumns, txStore.table()),
			params.SourceService, params.SourceService, limit,
		)
		if err != nil {
			return err
		}
		candidates := make([]storedEvent, 0, limit)
		for rows.Next() {
			row, err := scanEvent(rows.Scan)
			if err != nil {
				_ = rows.Close()
				return err
			}
			candidates = append(candidates, row)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}

		lockedBy := sql.NullString{String: workerID, Valid: true}
		lockedUntil := sql.NullTime{Time: time.Now().Add(leaseTimeout), Valid: true}
		for _, row := range candidates {
			affected, err := txStore.execRows(ctx, `
SET status = 'processing', locked_by = ?, locked_until = ?, updated_at = NOW()
WHERE id = ? AND status IN ('pending', 'processing')
  AND (locked_until IS NULL OR locked_until <= NOW())`,
				lockedBy, lockedUntil, row.ID,
			)
			if err != nil {
				return err
			}
			if affected == 0 {
				continue
			}
			row.Status = callbacksqlc.ClbEventStatusProcessing
			row.LockedBy = lockedBy
			row.LockedUntil = lockedUntil
			leased = append(leased, row)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return leased, nil
}

func (s *Store) MarkOK(ctx context.Context, id uint64, workerID string) error {
	rows, err := s.execRows(ctx, `
SET status = 'ok', delivered_at = NOW(), locked_by = NULL,
    locked_until = NULL, last_error = NULL, updated_at = NOW()
WHERE id = ? AND status = 'processing' AND locked_by = ?`,
		id, normalizeWorkerID(workerID),
	)
	return leasedResult(rows, err)
}

func (s *Store) MarkReject(ctx context.Context, id uint64, workerID string, reason string) error {
	rows, err := s.execRows(ctx, `
SET status = 'reject', rejected_at = NOW(), reject_reason = ?,
    locked_by = NULL, locked_until = NULL, updated_at = NOW()
WHERE id = ? AND status = 'processing' AND locked_by = ?`,
		nullableString(reason), id, normalizeWorkerID(workerID),
	)
	return leasedResult(rows, err)
}

func (s *Store) MarkFailed(ctx context.Context, params FailParams) error {
	failedAt := params.FailedAt
	if failedAt.IsZero() {
		failedAt = time.Now()
	}
	rows, err := s.execRows(ctx, `
SET status = 'pending', attempt_count = attempt_count + 1,
    next_attempt_at = ?, locked_by = NULL, locked_until = NULL,
    last_error = ?, updated_at = NOW()
WHERE id = ? AND status = 'processing' AND locked_by = ?`,
		failedAt.Add(RetryDelay(params.Attempt)), nullableString(params.Error),
		params.ID, normalizeWorkerID(params.WorkerID),
	)
	return leasedResult(rows, err)
}

func RetryDelay(attempt uint32) time.Duration {
	switch attempt {
	case 0:
		return 5 * time.Second
	case 1:
		return 30 * time.Second
	case 2:
		return time.Minute
	case 3:
		return 5 * time.Minute
	case 4:
		return 10 * time.Minute
	case 5:
		return 30 * time.Minute
	default:
		return time.Hour
	}
}

func normalizeWorkerID(workerID string) string {
	if workerID == "" {
		return "default"
	}
	return workerID
}

func normalizePage(limit int32, offset int32) (int32, int32) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func (s *Store) withTx(ctx context.Context, fn func(*Store) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txStore := s.WithTx(tx)
	if err := fn(txStore); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("%w: rollback: %v", err, rollbackErr)
		}
		return err
	}
	return tx.Commit()
}

func (s *Store) execRows(ctx context.Context, update string, args ...any) (int64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}
	result, err := s.executor.ExecContext(ctx, "UPDATE "+s.table()+" "+update, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) validate() error {
	if s == nil || s.db == nil || s.executor == nil {
		return ErrStoreNotConfigured
	}
	if !tableNameExpression.MatchString(s.tableName) {
		return errors.New("callback: invalid table name")
	}
	return nil
}

func (s *Store) table() string {
	return quoteIdentifier(s.tableName)
}

func normalizeTableName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if !tableNameExpression.MatchString(value) {
		return DefaultTable
	}
	return value
}

func quoteIdentifier(value string) string {
	return "`" + value + "`"
}

func nullableString(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func leasedResult(rows int64, err error) error {
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotLeased
	}
	return nil
}

const eventColumns = `
id, source_service, event_type, event_key, idempotency_key,
payload, payload_content_type, status, attempt_count, next_attempt_at,
locked_by, locked_until, delivered_at, rejected_at, last_error,
reject_reason, created_at, updated_at`

type scanFunc func(...any) error

type storedEvent struct {
	callbacksqlc.ClbEvent
}

func scanEvent(scan scanFunc) (storedEvent, error) {
	var value storedEvent
	err := scan(
		&value.ID,
		&value.SourceService,
		&value.EventType,
		&value.EventKey,
		&value.IdempotencyKey,
		&value.Payload,
		&value.PayloadContentType,
		&value.Status,
		&value.AttemptCount,
		&value.NextAttemptAt,
		&value.LockedBy,
		&value.LockedUntil,
		&value.DeliveredAt,
		&value.RejectedAt,
		&value.LastError,
		&value.RejectReason,
		&value.CreatedAt,
		&value.UpdatedAt,
	)
	if err != nil {
		return storedEvent{}, err
	}
	return value, nil
}

func mapEvent(value storedEvent) Event {
	return Event{
		ID:                 value.ID,
		SourceService:      value.SourceService,
		EventType:          value.EventType,
		EventKey:           value.EventKey,
		IdempotencyKey:     value.IdempotencyKey,
		Payload:            value.Payload,
		PayloadContentType: value.PayloadContentType,
		Status:             string(value.Status),
		AttemptCount:       value.AttemptCount,
		NextAttemptAt:      value.NextAttemptAt,
		LockedBy:           nullStringPtr(value.LockedBy),
		LockedUntil:        nullTimePtr(value.LockedUntil),
		DeliveredAt:        nullTimePtr(value.DeliveredAt),
		RejectedAt:         nullTimePtr(value.RejectedAt),
		LastError:          nullStringPtr(value.LastError),
		RejectReason:       nullStringPtr(value.RejectReason),
		CreatedAt:          value.CreatedAt,
		UpdatedAt:          value.UpdatedAt,
	}
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
