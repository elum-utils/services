DROP TRIGGER IF EXISTS calendar_operation_update_progress;
CREATE TRIGGER calendar_operation_update_progress
AFTER INSERT ON calendar_operation
FOR EACH ROW
INSERT INTO calendar_progress (
    workspace_id, calendar_id, app_id, platform_id, platform_user_id,
    current_position, claim_count, last_claim_position, last_claim_at,
    next_claim_at, is_completed, reset_count, last_was_reset
)
SELECT
    NEW.workspace_id, NEW.calendar_id, NEW.app_id, NEW.platform_id,
    NEW.platform_user_id, NEW.current_position, NEW.claim_count,
    NEW.last_claim_position, NEW.last_claim_at, NEW.next_claim_at,
    NEW.is_completed, NEW.reset_count, NEW.was_reset
WHERE NEW.granted = TRUE
ON DUPLICATE KEY UPDATE
    current_position = VALUES(current_position),
    claim_count = VALUES(claim_count),
    last_claim_position = VALUES(last_claim_position),
    last_claim_at = VALUES(last_claim_at),
    next_claim_at = VALUES(next_claim_at),
    is_completed = VALUES(is_completed),
    reset_count = VALUES(reset_count),
    last_was_reset = VALUES(last_was_reset);

DROP TRIGGER IF EXISTS calendar_operation_create_callback;
CREATE TRIGGER calendar_operation_create_callback
AFTER INSERT ON calendar_operation
FOR EACH ROW
INSERT INTO calendar_clb_event (
    source_service, event_type, event_key, idempotency_key, payload,
    payload_content_type, next_attempt_at
)
SELECT
    'calendar',
    'calendar.reward_granted',
    CONCAT('calendar.reward_granted:', NEW.id),
    CONCAT('calendar.reward_granted:', NEW.id),
    JSON_OBJECT(
        'operation_row_id', NEW.id,
        'operation_id', NEW.operation_id,
        'workspace_id', NEW.workspace_id,
        'calendar_id', NEW.calendar_id,
        'app_id', NEW.app_id,
        'platform_id', NEW.platform_id,
        'platform_user_id', NEW.platform_user_id,
        'position', NEW.position,
        'rewards', NEW.rewards_snapshot,
        'occurred_at', DATE_FORMAT(NEW.occurred_at, '%Y-%m-%dT%H:%i:%s.000000Z')
    ),
    'application/json',
    NOW()
WHERE NEW.granted = TRUE;
