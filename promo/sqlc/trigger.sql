DROP TRIGGER IF EXISTS promo_redemption_validate_limit;
CREATE TRIGGER promo_redemption_validate_limit
BEFORE INSERT ON promo_redemption
FOR EACH ROW
SET NEW.reward_snapshot = IF(
    EXISTS (
        SELECT 1
        FROM promo_offer o
        WHERE o.workspace_id = NEW.workspace_id
          AND o.id = NEW.promo_id
          AND o.deleted_at IS NULL
          AND (o.max_activations = 0 OR o.activation_count < o.max_activations)
    ),
    NEW.reward_snapshot,
    NULL
);

DROP TRIGGER IF EXISTS promo_redemption_increment_activation;
CREATE TRIGGER promo_redemption_increment_activation
AFTER INSERT ON promo_redemption
FOR EACH ROW
UPDATE promo_offer
SET activation_count = activation_count + 1
WHERE workspace_id = NEW.workspace_id
  AND id = NEW.promo_id;

DROP TRIGGER IF EXISTS promo_redemption_create_event;
CREATE TRIGGER promo_redemption_create_event
AFTER INSERT ON promo_redemption
FOR EACH ROW
INSERT INTO promo_redemption_event (
    workspace_id, promo_id, redemption_id
) VALUES (
    NEW.workspace_id, NEW.promo_id, NEW.id
);

DROP TRIGGER IF EXISTS promo_redemption_create_callback;
CREATE TRIGGER promo_redemption_create_callback
AFTER INSERT ON promo_redemption
FOR EACH ROW
INSERT INTO promo_clb_event (
    source_service,
    event_type,
    event_key,
    idempotency_key,
    payload,
    payload_content_type,
    next_attempt_at
)
SELECT
    'promo',
    'promo.applied',
    CONCAT('promo.applied:', NEW.id),
    CONCAT('promo.applied:', NEW.id),
    JSON_OBJECT(
        'redemption_id', NEW.id,
        'workspace_id', NEW.workspace_id,
        'promo_id', NEW.promo_id,
        'code', o.code,
        'app_id', NEW.app_id,
        'platform_id', NEW.platform_id,
        'platform_user_id', NEW.platform_user_id,
        'rewards', NEW.reward_snapshot
    ),
    'application/json',
    NOW()
FROM promo_offer o
WHERE o.workspace_id = NEW.workspace_id
  AND o.id = NEW.promo_id;
