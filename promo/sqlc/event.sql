CREATE EVENT IF NOT EXISTS promo_refresh_daily_stats
ON SCHEDULE
    EVERY '1' DAY
    STARTS '2025-11-08 00:05:00'
DO
INSERT INTO promo_stats_daily (
    workspace_id, promo_id, stats_date, redemption_count, unique_users
)
SELECT
    e.workspace_id,
    e.promo_id,
    DATE(e.occurred_at),
    COUNT(*),
    COUNT(DISTINCT CONCAT_WS(':', r.app_id, r.platform_id, r.platform_user_id))
FROM promo_redemption_event e
JOIN promo_redemption r
  ON r.workspace_id = e.workspace_id
 AND r.promo_id = e.promo_id
 AND r.id = e.redemption_id
WHERE e.occurred_at >= CURRENT_DATE - INTERVAL 2 DAY
GROUP BY e.workspace_id, e.promo_id, DATE(e.occurred_at)
ON DUPLICATE KEY UPDATE
    redemption_count = VALUES(redemption_count),
    unique_users = VALUES(unique_users),
    updated_at = NOW();
