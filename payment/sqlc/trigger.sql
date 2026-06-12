DROP TRIGGER IF EXISTS payment_order_create_purchase_stats;
CREATE TRIGGER payment_order_create_purchase_stats
AFTER UPDATE ON payment_order
FOR EACH ROW
INSERT IGNORE INTO payment_stats_event (
    event_type, source_id, workspace_id, product_id,
    app_id, platform_id, platform_user_id, quantity,
    asset_code, amount_minor, occurred_at
)
SELECT
    'purchase', NEW.id, NEW.workspace_id, NEW.product_id,
    NEW.app_id,
    COALESCE(NEW.payer_platform_id, NEW.platform_id),
    COALESCE(NEW.payer_platform_user_id, NEW.platform_user_id),
    NEW.quantity,
    NEW.asset_code, NEW.payable_amount_minor, COALESCE(NEW.fulfilled_at, NOW())
WHERE NEW.status = 'fulfilled'
  AND OLD.status <> 'fulfilled';

DROP TRIGGER IF EXISTS payment_refund_create_stats;
CREATE TRIGGER payment_refund_create_stats
AFTER INSERT ON payment_refund
FOR EACH ROW
INSERT IGNORE INTO payment_stats_event (
    event_type, source_id, workspace_id, product_id,
    app_id, platform_id, platform_user_id, quantity,
    asset_code, amount_minor, occurred_at
)
SELECT
    'refund', NEW.id, o.workspace_id, o.product_id,
    o.app_id,
    COALESCE(o.payer_platform_id, o.platform_id),
    COALESCE(o.payer_platform_user_id, o.platform_user_id),
    0,
    NEW.asset_code, NEW.amount_minor, NEW.created_at
FROM payment_order o
WHERE o.id = NEW.order_id
  AND NEW.status = 'succeeded';

DROP TRIGGER IF EXISTS payment_refund_update_stats;
CREATE TRIGGER payment_refund_update_stats
AFTER UPDATE ON payment_refund
FOR EACH ROW
INSERT IGNORE INTO payment_stats_event (
    event_type, source_id, workspace_id, product_id,
    app_id, platform_id, platform_user_id, quantity,
    asset_code, amount_minor, occurred_at
)
SELECT
    'refund', NEW.id, o.workspace_id, o.product_id,
    o.app_id,
    COALESCE(o.payer_platform_id, o.platform_id),
    COALESCE(o.payer_platform_user_id, o.platform_user_id),
    0,
    NEW.asset_code, NEW.amount_minor, NEW.updated_at
FROM payment_order o
WHERE o.id = NEW.order_id
  AND NEW.status = 'succeeded'
  AND OLD.status <> 'succeeded';

DROP TRIGGER IF EXISTS payment_order_create_daily_stats;
CREATE TRIGGER payment_order_create_daily_stats
AFTER INSERT ON payment_order
FOR EACH ROW
INSERT IGNORE INTO payment_stats_order_event (
    order_id, workspace_id, product_id,
    event_type, order_status, occurred_at
)
SELECT
    NEW.id, NEW.workspace_id, NEW.product_id,
    'created', NEW.status, NEW.created_at
UNION ALL
SELECT
    NEW.id, NEW.workspace_id, NEW.product_id,
    'status', NEW.status, NEW.created_at;

DROP TRIGGER IF EXISTS payment_order_update_daily_stats;
CREATE TRIGGER payment_order_update_daily_stats
AFTER UPDATE ON payment_order
FOR EACH ROW
INSERT IGNORE INTO payment_stats_order_event (
    order_id, workspace_id, product_id,
    event_type, order_status, occurred_at
)
SELECT
    NEW.id, NEW.workspace_id, NEW.product_id,
    'status', NEW.status, NEW.updated_at
WHERE NEW.status <> OLD.status;

DROP TRIGGER IF EXISTS payment_order_event_update_daily_overview;
CREATE TRIGGER payment_order_event_update_daily_overview
AFTER INSERT ON payment_stats_order_event
FOR EACH ROW
INSERT INTO payment_stats_daily_overview (
    workspace_id,
    stats_date,
    orders_created,
    draft_orders,
    pending_payment_orders,
    paid_orders,
    fulfilled_orders,
    canceled_orders,
    expired_orders,
    refunded_orders,
    chargebacked_orders,
    failed_orders
)
VALUES (
    NEW.workspace_id,
    DATE(NEW.occurred_at),
    NEW.event_type = 'created',
    NEW.event_type = 'status' AND NEW.order_status = 'draft',
    NEW.event_type = 'status' AND NEW.order_status = 'pending_payment',
    NEW.event_type = 'status' AND NEW.order_status = 'paid',
    NEW.event_type = 'status' AND NEW.order_status = 'fulfilled',
    NEW.event_type = 'status' AND NEW.order_status = 'canceled',
    NEW.event_type = 'status' AND NEW.order_status = 'expired',
    NEW.event_type = 'status' AND NEW.order_status = 'refunded',
    NEW.event_type = 'status' AND NEW.order_status = 'chargebacked',
    NEW.event_type = 'status' AND NEW.order_status = 'failed'
)
ON DUPLICATE KEY UPDATE
    orders_created = orders_created + VALUES(orders_created),
    draft_orders = draft_orders + VALUES(draft_orders),
    pending_payment_orders = pending_payment_orders + VALUES(pending_payment_orders),
    paid_orders = paid_orders + VALUES(paid_orders),
    fulfilled_orders = fulfilled_orders + VALUES(fulfilled_orders),
    canceled_orders = canceled_orders + VALUES(canceled_orders),
    expired_orders = expired_orders + VALUES(expired_orders),
    refunded_orders = refunded_orders + VALUES(refunded_orders),
    chargebacked_orders = chargebacked_orders + VALUES(chargebacked_orders),
    failed_orders = failed_orders + VALUES(failed_orders),
    updated_at = NOW();

DROP TRIGGER IF EXISTS payment_stats_event_update_daily_overview;
CREATE TRIGGER payment_stats_event_update_daily_overview
AFTER INSERT ON payment_stats_event
FOR EACH ROW
INSERT INTO payment_stats_daily_overview (
    workspace_id,
    stats_date,
    purchase_count,
    purchase_quantity,
    refund_count
)
VALUES (
    NEW.workspace_id,
    DATE(NEW.occurred_at),
    NEW.event_type = 'purchase',
    IF(NEW.event_type = 'purchase', NEW.quantity, 0),
    NEW.event_type = 'refund'
)
ON DUPLICATE KEY UPDATE
    purchase_count = purchase_count + VALUES(purchase_count),
    purchase_quantity = purchase_quantity + VALUES(purchase_quantity),
    refund_count = refund_count + VALUES(refund_count),
    updated_at = NOW();

DROP TRIGGER IF EXISTS payment_stats_event_create_daily_buyer;
CREATE TRIGGER payment_stats_event_create_daily_buyer
AFTER INSERT ON payment_stats_event
FOR EACH ROW
INSERT IGNORE INTO payment_stats_daily_buyer (
    workspace_id,
    stats_date,
    app_id,
    platform_id,
    platform_user_id
)
SELECT
    NEW.workspace_id,
    DATE(NEW.occurred_at),
    NEW.app_id,
    NEW.platform_id,
    NEW.platform_user_id
WHERE NEW.event_type = 'purchase';

DROP TRIGGER IF EXISTS payment_daily_buyer_update_overview;
CREATE TRIGGER payment_daily_buyer_update_overview
AFTER INSERT ON payment_stats_daily_buyer
FOR EACH ROW
INSERT INTO payment_stats_daily_overview (
    workspace_id,
    stats_date,
    unique_buyers
)
VALUES (
    NEW.workspace_id,
    NEW.stats_date,
    1
)
ON DUPLICATE KEY UPDATE
    unique_buyers = unique_buyers + 1,
    updated_at = NOW();

DROP TRIGGER IF EXISTS payment_product_create_daily_overview;
DROP TRIGGER IF EXISTS payment_product_update_daily_overview;
DROP TRIGGER IF EXISTS payment_product_delete_daily_overview;

INSERT IGNORE INTO payment_stats_event (
    event_type, source_id, workspace_id, product_id,
    app_id, platform_id, platform_user_id, quantity,
    asset_code, amount_minor, occurred_at
)
SELECT
    'purchase', o.id, o.workspace_id, o.product_id,
    o.app_id,
    COALESCE(o.payer_platform_id, o.platform_id),
    COALESCE(o.payer_platform_user_id, o.platform_user_id),
    o.quantity,
    o.asset_code, o.payable_amount_minor,
    COALESCE(f.fulfilled_at, o.fulfilled_at, f.created_at)
FROM payment_order o
JOIN payment_fulfillment f ON f.order_id = o.id
WHERE f.status IN ('succeeded', 'revoked');

INSERT IGNORE INTO payment_stats_event (
    event_type, source_id, workspace_id, product_id,
    app_id, platform_id, platform_user_id, quantity,
    asset_code, amount_minor, occurred_at
)
SELECT
    'refund', r.id, o.workspace_id, o.product_id,
    o.app_id,
    COALESCE(o.payer_platform_id, o.platform_id),
    COALESCE(o.payer_platform_user_id, o.platform_user_id),
    0,
    r.asset_code, r.amount_minor, r.updated_at
FROM payment_refund r
JOIN payment_order o ON o.id = r.order_id
WHERE r.status = 'succeeded';

INSERT IGNORE INTO payment_stats_order_event (
    order_id, workspace_id, product_id,
    event_type, order_status, occurred_at
)
SELECT
    o.id, o.workspace_id, o.product_id,
    'created', o.status, o.created_at
FROM payment_order o
UNION ALL
SELECT
    o.id, o.workspace_id, o.product_id,
    'status', o.status, o.updated_at
FROM payment_order o;

INSERT IGNORE INTO payment_stats_daily_buyer (
    workspace_id,
    stats_date,
    app_id,
    platform_id,
    platform_user_id
)
SELECT
    workspace_id,
    DATE(occurred_at),
    app_id,
    platform_id,
    platform_user_id
FROM payment_stats_event
WHERE event_type = 'purchase';

INSERT INTO payment_stats_daily_overview (
    workspace_id,
    stats_date,
    purchase_count,
    purchase_quantity,
    unique_buyers,
    refund_count
)
SELECT
    workspace_id,
    DATE(occurred_at),
    SUM(event_type = 'purchase'),
    SUM(IF(event_type = 'purchase', quantity, 0)),
    COUNT(DISTINCT IF(
        event_type = 'purchase',
        CONCAT_WS(':', app_id, platform_id, platform_user_id),
        NULL
    )),
    SUM(event_type = 'refund')
FROM payment_stats_event
GROUP BY workspace_id, DATE(occurred_at)
ON DUPLICATE KEY UPDATE
    purchase_count = VALUES(purchase_count),
    purchase_quantity = VALUES(purchase_quantity),
    unique_buyers = VALUES(unique_buyers),
    refund_count = VALUES(refund_count),
    updated_at = NOW();

INSERT INTO payment_stats_daily_overview (
    workspace_id,
    stats_date,
    products_total,
    active_products,
    visible_products
)
SELECT
    workspace_id,
    CURRENT_DATE,
    COUNT(*),
    SUM(
        is_closed = FALSE
        AND available_from <= NOW()
        AND available_until > NOW()
    ),
    SUM(
        is_visible = TRUE
        AND is_closed = FALSE
        AND available_from <= NOW()
        AND available_until > NOW()
    )
FROM payment_product
GROUP BY workspace_id
ON DUPLICATE KEY UPDATE
    products_total = VALUES(products_total),
    active_products = VALUES(active_products),
    visible_products = VALUES(visible_products),
    updated_at = NOW();
