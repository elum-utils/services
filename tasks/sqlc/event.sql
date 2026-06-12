DROP EVENT IF EXISTS task_refresh_daily_stats;
CREATE EVENT task_refresh_daily_stats
ON SCHEDULE
    EVERY '1' DAY
    STARTS '2025-11-08 00:05:00'
DO
INSERT INTO task_stats_daily (
    workspace_id,
    task_id,
    stats_date,
    progress_created,
    progress_amount,
    ready_count,
    claimed_count,
    manual_claimed_count,
    auto_claimed_count,
    unique_participants,
    unique_claimers
)
SELECT
    workspace_id,
    task_id,
    DATE(occurred_at),
    SUM(event_type = 'progress_created'),
    SUM(IF(event_type = 'progress_added', amount, 0)),
    SUM(event_type = 'ready'),
    SUM(event_type = 'claimed'),
    SUM(event_type = 'claimed' AND claim_mode = 'manual'),
    SUM(event_type = 'claimed' AND claim_mode = 'auto'),
    COUNT(DISTINCT CONCAT_WS(':', app_id, platform_id, platform_user_id)),
    COUNT(DISTINCT IF(
        event_type = 'claimed',
        CONCAT_WS(':', app_id, platform_id, platform_user_id),
        NULL
    ))
FROM task_stats_event
WHERE occurred_at >= CURRENT_DATE - INTERVAL 2 DAY
GROUP BY workspace_id, task_id, DATE(occurred_at)
ON DUPLICATE KEY UPDATE
    progress_created = VALUES(progress_created),
    progress_amount = VALUES(progress_amount),
    ready_count = VALUES(ready_count),
    claimed_count = VALUES(claimed_count),
    manual_claimed_count = VALUES(manual_claimed_count),
    auto_claimed_count = VALUES(auto_claimed_count),
    unique_participants = VALUES(unique_participants),
    unique_claimers = VALUES(unique_claimers),
    updated_at = NOW();

DROP EVENT IF EXISTS task_refresh_daily_overview;
CREATE EVENT task_refresh_daily_overview
ON SCHEDULE
    EVERY '1' DAY
    STARTS '2025-11-08 00:05:00'
DO
INSERT INTO task_stats_daily_overview (
    workspace_id,
    stats_date,
    tasks_total,
    active_tasks,
    visible_tasks,
    progress_created,
    progress_amount,
    ready_count,
    claimed_count,
    manual_claimed_count,
    auto_claimed_count,
    unique_participants,
    unique_claimers
)
SELECT
    workspaces.workspace_id,
    dates.stats_date,
    COALESCE(definitions.tasks_total, 0),
    COALESCE(definitions.active_tasks, 0),
    COALESCE(definitions.visible_tasks, 0),
    COALESCE(events.progress_created, 0),
    COALESCE(events.progress_amount, 0),
    COALESCE(events.ready_count, 0),
    COALESCE(events.claimed_count, 0),
    COALESCE(events.manual_claimed_count, 0),
    COALESCE(events.auto_claimed_count, 0),
    COALESCE(events.unique_participants, 0),
    COALESCE(events.unique_claimers, 0)
FROM (
    SELECT workspace_id FROM task_definition
    UNION
    SELECT workspace_id FROM task_stats_event
) workspaces
CROSS JOIN (
    SELECT CURRENT_DATE - INTERVAL 2 DAY AS stats_date
    UNION ALL SELECT CURRENT_DATE - INTERVAL 1 DAY
    UNION ALL SELECT CURRENT_DATE
) dates
LEFT JOIN (
    SELECT
        workspace_id,
        COUNT(*) AS tasks_total,
        SUM(
            is_active = TRUE
            AND deleted_at IS NULL
            AND (start_at IS NULL OR start_at <= NOW())
            AND (end_at IS NULL OR end_at > NOW())
        ) AS active_tasks,
        SUM(
            is_visible = TRUE
            AND is_active = TRUE
            AND deleted_at IS NULL
            AND (start_at IS NULL OR start_at <= NOW())
            AND (end_at IS NULL OR end_at > NOW())
        ) AS visible_tasks
    FROM task_definition
    GROUP BY workspace_id
) definitions ON definitions.workspace_id = workspaces.workspace_id
LEFT JOIN (
    SELECT
        workspace_id,
        DATE(occurred_at) AS stats_date,
        SUM(event_type = 'progress_created') AS progress_created,
        SUM(IF(event_type = 'progress_added', amount, 0)) AS progress_amount,
        SUM(event_type = 'ready') AS ready_count,
        SUM(event_type = 'claimed') AS claimed_count,
        SUM(event_type = 'claimed' AND claim_mode = 'manual') AS manual_claimed_count,
        SUM(event_type = 'claimed' AND claim_mode = 'auto') AS auto_claimed_count,
        COUNT(DISTINCT CONCAT_WS(':', app_id, platform_id, platform_user_id)) AS unique_participants,
        COUNT(DISTINCT IF(
            event_type = 'claimed',
            CONCAT_WS(':', app_id, platform_id, platform_user_id),
            NULL
        )) AS unique_claimers
    FROM task_stats_event
    WHERE occurred_at >= CURRENT_DATE - INTERVAL 2 DAY
    GROUP BY workspace_id, DATE(occurred_at)
) events
    ON events.workspace_id = workspaces.workspace_id
   AND events.stats_date = dates.stats_date
WHERE TRUE
ON DUPLICATE KEY UPDATE
    tasks_total = IF(
        stats_date >= CURRENT_DATE - INTERVAL 1 DAY,
        VALUES(tasks_total),
        tasks_total
    ),
    active_tasks = IF(
        stats_date >= CURRENT_DATE - INTERVAL 1 DAY,
        VALUES(active_tasks),
        active_tasks
    ),
    visible_tasks = IF(
        stats_date >= CURRENT_DATE - INTERVAL 1 DAY,
        VALUES(visible_tasks),
        visible_tasks
    ),
    progress_created = VALUES(progress_created),
    progress_amount = VALUES(progress_amount),
    ready_count = VALUES(ready_count),
    claimed_count = VALUES(claimed_count),
    manual_claimed_count = VALUES(manual_claimed_count),
    auto_claimed_count = VALUES(auto_claimed_count),
    unique_participants = VALUES(unique_participants),
    unique_claimers = VALUES(unique_claimers),
    updated_at = NOW();
