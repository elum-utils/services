DROP EVENT IF EXISTS payment_refresh_daily_stats;
CREATE EVENT payment_refresh_daily_stats
ON SCHEDULE
    EVERY '1' DAY
    STARTS '2025-11-08 00:05:00'
DO
INSERT INTO payment_stats_daily (
    workspace_id, product_id, asset_code, stats_date,
    purchase_count, purchase_quantity, unique_buyers,
    gross_amount_minor, refund_count, refund_amount_minor
)
SELECT
    workspace_id,
    product_id,
    asset_code,
    DATE(occurred_at),
    SUM(event_type = 'purchase'),
    SUM(IF(event_type = 'purchase', quantity, 0)),
    COUNT(DISTINCT IF(
        event_type = 'purchase',
        CONCAT_WS(':', app_id, platform_id, platform_user_id),
        NULL
    )),
    SUM(IF(event_type = 'purchase', amount_minor, 0)),
    SUM(event_type = 'refund'),
    SUM(IF(event_type = 'refund', amount_minor, 0))
FROM payment_stats_event
WHERE occurred_at >= CURRENT_DATE - INTERVAL 2 DAY
GROUP BY workspace_id, product_id, asset_code, DATE(occurred_at)
UNION ALL
SELECT
    workspace_id,
    '',
    asset_code,
    DATE(occurred_at),
    SUM(event_type = 'purchase'),
    SUM(IF(event_type = 'purchase', quantity, 0)),
    COUNT(DISTINCT IF(
        event_type = 'purchase',
        CONCAT_WS(':', app_id, platform_id, platform_user_id),
        NULL
    )),
    SUM(IF(event_type = 'purchase', amount_minor, 0)),
    SUM(event_type = 'refund'),
    SUM(IF(event_type = 'refund', amount_minor, 0))
FROM payment_stats_event
WHERE occurred_at >= CURRENT_DATE - INTERVAL 2 DAY
GROUP BY workspace_id, asset_code, DATE(occurred_at)
ON DUPLICATE KEY UPDATE
    purchase_count = VALUES(purchase_count),
    purchase_quantity = VALUES(purchase_quantity),
    unique_buyers = VALUES(unique_buyers),
    gross_amount_minor = VALUES(gross_amount_minor),
    refund_count = VALUES(refund_count),
    refund_amount_minor = VALUES(refund_amount_minor),
    updated_at = NOW();

DROP EVENT IF EXISTS payment_refresh_daily_overview;
CREATE EVENT payment_refresh_daily_overview
ON SCHEDULE
    EVERY '1' DAY
    STARTS '2025-11-08 00:05:00'
DO
INSERT INTO payment_stats_daily_overview (
    workspace_id,
    stats_date,
    products_total,
    active_products,
    visible_products,
    orders_created,
    draft_orders,
    pending_payment_orders,
    paid_orders,
    fulfilled_orders,
    canceled_orders,
    expired_orders,
    refunded_orders,
    chargebacked_orders,
    failed_orders,
    purchase_count,
    purchase_quantity,
    unique_buyers,
    refund_count
)
SELECT
    workspaces.workspace_id,
    dates.stats_date,
    COALESCE(products.products_total, 0),
    COALESCE(products.active_products, 0),
    COALESCE(products.visible_products, 0),
    COALESCE(orders.orders_created, 0),
    COALESCE(orders.draft_orders, 0),
    COALESCE(orders.pending_payment_orders, 0),
    COALESCE(orders.paid_orders, 0),
    COALESCE(orders.fulfilled_orders, 0),
    COALESCE(orders.canceled_orders, 0),
    COALESCE(orders.expired_orders, 0),
    COALESCE(orders.refunded_orders, 0),
    COALESCE(orders.chargebacked_orders, 0),
    COALESCE(orders.failed_orders, 0),
    COALESCE(payments.purchase_count, 0),
    COALESCE(payments.purchase_quantity, 0),
    COALESCE(payments.unique_buyers, 0),
    COALESCE(payments.refund_count, 0)
FROM (
    SELECT workspace_id FROM payment_product
    UNION
    SELECT workspace_id FROM payment_order
    UNION
    SELECT workspace_id FROM payment_stats_event
) workspaces
CROSS JOIN (
    SELECT CURRENT_DATE - INTERVAL 2 DAY AS stats_date
    UNION ALL SELECT CURRENT_DATE - INTERVAL 1 DAY
    UNION ALL SELECT CURRENT_DATE
) dates
LEFT JOIN (
    SELECT
        workspace_id,
        COUNT(*) AS products_total,
        SUM(
            is_closed = FALSE
            AND available_from <= NOW()
            AND available_until > NOW()
        ) AS active_products,
        SUM(
            is_visible = TRUE
            AND is_closed = FALSE
            AND available_from <= NOW()
            AND available_until > NOW()
        ) AS visible_products
    FROM payment_product
    GROUP BY workspace_id
) products ON products.workspace_id = workspaces.workspace_id
LEFT JOIN (
    SELECT
        workspace_id,
        DATE(occurred_at) AS stats_date,
        SUM(event_type = 'created') AS orders_created,
        SUM(event_type = 'status' AND order_status = 'draft') AS draft_orders,
        SUM(event_type = 'status' AND order_status = 'pending_payment') AS pending_payment_orders,
        SUM(event_type = 'status' AND order_status = 'paid') AS paid_orders,
        SUM(event_type = 'status' AND order_status = 'fulfilled') AS fulfilled_orders,
        SUM(event_type = 'status' AND order_status = 'canceled') AS canceled_orders,
        SUM(event_type = 'status' AND order_status = 'expired') AS expired_orders,
        SUM(event_type = 'status' AND order_status = 'refunded') AS refunded_orders,
        SUM(event_type = 'status' AND order_status = 'chargebacked') AS chargebacked_orders,
        SUM(event_type = 'status' AND order_status = 'failed') AS failed_orders
    FROM payment_stats_order_event
    WHERE occurred_at >= CURRENT_DATE - INTERVAL 2 DAY
    GROUP BY workspace_id, DATE(occurred_at)
) orders
    ON orders.workspace_id = workspaces.workspace_id
   AND orders.stats_date = dates.stats_date
LEFT JOIN (
    SELECT
        workspace_id,
        DATE(occurred_at) AS stats_date,
        SUM(event_type = 'purchase') AS purchase_count,
        SUM(IF(event_type = 'purchase', quantity, 0)) AS purchase_quantity,
        COUNT(DISTINCT IF(
            event_type = 'purchase',
            CONCAT_WS(':', app_id, platform_id, platform_user_id),
            NULL
        )) AS unique_buyers,
        SUM(event_type = 'refund') AS refund_count
    FROM payment_stats_event
    WHERE occurred_at >= CURRENT_DATE - INTERVAL 2 DAY
    GROUP BY workspace_id, DATE(occurred_at)
) payments
    ON payments.workspace_id = workspaces.workspace_id
   AND payments.stats_date = dates.stats_date
WHERE TRUE
ON DUPLICATE KEY UPDATE
    products_total = IF(
        stats_date >= CURRENT_DATE - INTERVAL 1 DAY,
        VALUES(products_total),
        products_total
    ),
    active_products = IF(
        stats_date >= CURRENT_DATE - INTERVAL 1 DAY,
        VALUES(active_products),
        active_products
    ),
    visible_products = IF(
        stats_date >= CURRENT_DATE - INTERVAL 1 DAY,
        VALUES(visible_products),
        visible_products
    ),
    orders_created = VALUES(orders_created),
    draft_orders = VALUES(draft_orders),
    pending_payment_orders = VALUES(pending_payment_orders),
    paid_orders = VALUES(paid_orders),
    fulfilled_orders = VALUES(fulfilled_orders),
    canceled_orders = VALUES(canceled_orders),
    expired_orders = VALUES(expired_orders),
    refunded_orders = VALUES(refunded_orders),
    chargebacked_orders = VALUES(chargebacked_orders),
    failed_orders = VALUES(failed_orders),
    purchase_count = VALUES(purchase_count),
    purchase_quantity = VALUES(purchase_quantity),
    unique_buyers = VALUES(unique_buyers),
    refund_count = VALUES(refund_count),
    updated_at = NOW();
