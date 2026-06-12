CREATE EVENT IF NOT EXISTS cpa_refresh_daily_stats
ON SCHEDULE
    EVERY '1' DAY
    STARTS '2025-11-08 00:05:00'
DO
INSERT INTO cpa_stats_daily (
    workspace_id,
    cpa_id,
    stats_date,
    issued_count,
    completed_count,
    unique_users
)
SELECT
    workspace_id,
    cpa_id,
    DATE(occurred_at),
    SUM(event_type = 'issued'),
    SUM(event_type = 'completed'),
    COUNT(DISTINCT assignment_id)
FROM cpa_assignment_event
WHERE occurred_at >= CURRENT_DATE - INTERVAL 2 DAY
GROUP BY workspace_id, cpa_id, DATE(occurred_at)
ON DUPLICATE KEY UPDATE
    issued_count = VALUES(issued_count),
    completed_count = VALUES(completed_count),
    unique_users = VALUES(unique_users),
    updated_at = NOW();
