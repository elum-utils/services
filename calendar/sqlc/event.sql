CREATE EVENT IF NOT EXISTS calendar_refresh_daily_stats
ON SCHEDULE
    EVERY '1' DAY
    STARTS '2025-11-08 00:05:00'
DO
INSERT INTO calendar_stats_daily (
    workspace_id, calendar_id, stats_date,
    operation_count, grant_count, unique_users
)
SELECT
    workspace_id,
    calendar_id,
    DATE(occurred_at),
    COUNT(*),
    COALESCE(SUM(granted), 0),
    COUNT(DISTINCT CONCAT_WS(':', app_id, platform_id, platform_user_id))
FROM calendar_operation
WHERE occurred_at >= CURRENT_DATE - INTERVAL 2 DAY
GROUP BY workspace_id, calendar_id, DATE(occurred_at)
ON DUPLICATE KEY UPDATE
    operation_count = VALUES(operation_count),
    grant_count = VALUES(grant_count),
    unique_users = VALUES(unique_users),
    updated_at = NOW();
