-- name: CreateEvent :execlastid
INSERT INTO clb_event (
    source_service,
    event_type,
    event_key,
    idempotency_key,
    payload,
    payload_content_type,
    next_attempt_at
)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    id = LAST_INSERT_ID(id);

-- name: GetEvent :one
SELECT
    id,
    source_service,
    event_type,
    event_key,
    idempotency_key,
    payload,
    payload_content_type,
    status,
    attempt_count,
    next_attempt_at,
    locked_by,
    locked_until,
    delivered_at,
    rejected_at,
    last_error,
    reject_reason,
    created_at,
    updated_at
FROM clb_event
WHERE id = ?
LIMIT 1;

-- name: ListDueEventsForUpdate :many
SELECT
    id,
    source_service,
    event_type,
    event_key,
    idempotency_key,
    payload,
    payload_content_type,
    status,
    attempt_count,
    next_attempt_at,
    locked_by,
    locked_until,
    delivered_at,
    rejected_at,
    last_error,
    reject_reason,
    created_at,
    updated_at
FROM clb_event
WHERE (? = '' OR source_service = ?)
  AND status IN ('pending', 'processing')
  AND next_attempt_at <= NOW()
  AND (locked_until IS NULL OR locked_until <= NOW())
ORDER BY next_attempt_at, id
LIMIT ?
FOR UPDATE SKIP LOCKED;

-- name: MarkEventProcessing :execrows
UPDATE clb_event
SET status = 'processing',
    locked_by = ?,
    locked_until = ?,
    updated_at = NOW()
WHERE id = ?
  AND status IN ('pending', 'processing')
  AND (locked_until IS NULL OR locked_until <= NOW());

-- name: MarkEventOK :execrows
UPDATE clb_event
SET status = 'ok',
    delivered_at = NOW(),
    locked_by = NULL,
    locked_until = NULL,
    last_error = NULL,
    updated_at = NOW()
WHERE id = ?
  AND status = 'processing'
  AND locked_by = ?;

-- name: MarkEventReject :execrows
UPDATE clb_event
SET status = 'reject',
    rejected_at = NOW(),
    reject_reason = ?,
    locked_by = NULL,
    locked_until = NULL,
    updated_at = NOW()
WHERE id = ?
  AND status = 'processing'
  AND locked_by = ?;

-- name: MarkEventFailed :execrows
UPDATE clb_event
SET status = 'pending',
    attempt_count = attempt_count + 1,
    next_attempt_at = ?,
    locked_by = NULL,
    locked_until = NULL,
    last_error = ?,
    updated_at = NOW()
WHERE id = ?
  AND status = 'processing'
  AND locked_by = ?;

-- name: AdminListEvents :many
SELECT
    id,
    source_service,
    event_type,
    event_key,
    idempotency_key,
    payload,
    payload_content_type,
    status,
    attempt_count,
    next_attempt_at,
    locked_by,
    locked_until,
    delivered_at,
    rejected_at,
    last_error,
    reject_reason,
    created_at,
    updated_at
FROM clb_event
WHERE (? = '' OR source_service = ?)
  AND (? = '' OR event_type = ?)
  AND (? = '' OR CAST(status AS CHAR) = ?)
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: AdminRetryEventNow :execrows
UPDATE clb_event
SET status = 'pending',
    next_attempt_at = NOW(),
    locked_by = NULL,
    locked_until = NULL,
    last_error = NULL,
    updated_at = NOW()
WHERE id = ?
  AND status IN ('pending', 'processing');

-- name: AdminMarkEventOK :execrows
UPDATE clb_event
SET status = 'ok',
    delivered_at = NOW(),
    locked_by = NULL,
    locked_until = NULL,
    last_error = NULL,
    updated_at = NOW()
WHERE id = ?
  AND status IN ('pending', 'processing');

-- name: AdminMarkEventReject :execrows
UPDATE clb_event
SET status = 'reject',
    rejected_at = NOW(),
    reject_reason = ?,
    locked_by = NULL,
    locked_until = NULL,
    updated_at = NOW()
WHERE id = ?
  AND status IN ('pending', 'processing');

-- name: AdminResetExpiredProcessing :execrows
UPDATE clb_event
SET status = 'pending',
    locked_by = NULL,
    locked_until = NULL,
    next_attempt_at = NOW(),
    updated_at = NOW()
WHERE status = 'processing'
  AND locked_until IS NOT NULL
  AND locked_until <= NOW();
