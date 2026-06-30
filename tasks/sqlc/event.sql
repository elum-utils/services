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

DROP EVENT IF EXISTS task_partner_refresh_daily_stats;
CREATE EVENT task_partner_refresh_daily_stats
ON SCHEDULE
    EVERY '1' DAY
    STARTS '2025-11-08 00:07:00'
DO
INSERT INTO task_partner_stats_daily (
    workspace_id,
    stats_date,
    provider,
    group_key,
    external_type,
    issued_count,
    completed_count,
    claimed_count,
    revoked_count,
    revoked_after_claim_count,
    failed_count,
    fake_count,
    expired_count,
    unique_issued_users,
    unique_completed_users,
    unique_claimers
)
SELECT
    event_counts.workspace_id,
    event_counts.stats_date,
    event_counts.provider,
    event_counts.group_key,
    event_counts.external_type,
    event_counts.issued_count,
    event_counts.completed_count,
    event_counts.claimed_count,
    event_counts.revoked_count,
    event_counts.revoked_after_claim_count,
    event_counts.failed_count,
    event_counts.fake_count,
    event_counts.expired_count,
    COALESCE(unique_counts.unique_issued_users, 0),
    COALESCE(unique_counts.unique_completed_users, 0),
    COALESCE(unique_counts.unique_claimers, 0)
FROM (
    SELECT
        workspace_id,
        DATE(occurred_at) AS stats_date,
        provider,
        group_key,
        external_type,
        SUM(event_type = 'issued') AS issued_count,
        SUM(event_type = 'completed') AS completed_count,
        SUM(event_type = 'claimed') AS claimed_count,
        SUM(event_type = 'revoked') AS revoked_count,
        SUM(event_type = 'revoked_after_claim') AS revoked_after_claim_count,
        SUM(event_type = 'failed') AS failed_count,
        SUM(event_type = 'fake' OR status IN ('fake', 'fraud_suspected')) AS fake_count,
        SUM(event_type = 'expired' OR status IN ('expired', 'offer_expired')) AS expired_count
    FROM task_partner_stats_event
    WHERE occurred_at >= CURRENT_DATE - INTERVAL 2 DAY
    GROUP BY workspace_id, DATE(occurred_at), provider, group_key, external_type
) event_counts
LEFT JOIN (
    SELECT
        workspace_id,
        stats_date,
        provider,
        group_key,
        external_type,
        SUM(event_type = 'issued') AS unique_issued_users,
        SUM(event_type = 'completed') AS unique_completed_users,
        SUM(event_type = 'claimed') AS unique_claimers
    FROM task_partner_stats_unique_user
    WHERE stats_date >= CURRENT_DATE - INTERVAL 2 DAY
    GROUP BY workspace_id, stats_date, provider, group_key, external_type
) unique_counts
    ON unique_counts.workspace_id = event_counts.workspace_id
   AND unique_counts.stats_date = event_counts.stats_date
   AND unique_counts.provider = event_counts.provider
   AND unique_counts.group_key = event_counts.group_key
   AND unique_counts.external_type = event_counts.external_type
ON DUPLICATE KEY UPDATE
    issued_count = VALUES(issued_count),
    completed_count = VALUES(completed_count),
    claimed_count = VALUES(claimed_count),
    revoked_count = VALUES(revoked_count),
    revoked_after_claim_count = VALUES(revoked_after_claim_count),
    failed_count = VALUES(failed_count),
    fake_count = VALUES(fake_count),
    expired_count = VALUES(expired_count),
    unique_issued_users = VALUES(unique_issued_users),
    unique_completed_users = VALUES(unique_completed_users),
    unique_claimers = VALUES(unique_claimers),
    updated_at = NOW();
