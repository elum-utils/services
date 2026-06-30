# Tasks methods

Методы слоев `user` и `admin`, которые можно использовать как основу будущего API. Внутренние методы вынесены отдельно: они нужны контроллерам webhook/postback и не вызываются фронтом напрямую.

## user

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `User.ListActive(ctx, params)` | `ListActiveParams{Identity, Locale, GroupKey, Now}`; `Identity{WorkspaceID, AppID, PlatformID, Platform, PlatformUserID, IsPremium, Sex, Country}`. | Возвращает активные задачи пользователя сгруппированными как `[]TaskGroupModel{Key, Title, Description, Tasks}`. Если передан `GroupKey`, возвращает только эту группу. Группа и задачи локализуются по `Locale`; прогресс остается внутри каждой задачи. |
| `User.StartTask(ctx, params)` | `StartTaskParams{Identity, TaskRef, Now}`. | Берет задачу в работу. Для обычных задач создает текущий `progress=open`; для partner issue делегирует в `StartPartner`. Если `start_mode=required`, выполнение/claim до старта возвращают `not_started`. |
| `User.ListPartner(ctx, params)` | `PartnerListParams{Identity, Provider, GroupKey, Platform, Locale, Limit, Variables, Now}`. | Запрашивает партнерскую группу, скрыто вызывает provider adapter, создает/переиспользует `partner_issue` и возвращает партнерские задания в той же форме, что обычные задачи. |
| `User.StartPartner(ctx, params)` | `PartnerStartParams{Identity, IssueRef, Variables, Now}`. | Открывает/стартует партнерское задание, если партнеру нужен отдельный click/tracking шаг. Для GetBonus генерирует/сохраняет `external_click_id`, вызывает partner runtime и возвращает `action_url`; обычный claim не меняется. |
| `User.CheckPartner(ctx, params)` | `PartnerCheckParams{Identity, IssueRef, Variables, Now}`. | Проверяет выполнение партнерского задания через provider adapter, скрывает партнерский API и при успехе помечает issue как `completed`, возвращая задачу со статусом `ready`. |
| `User.Claim(ctx, params)` | `ClaimParams{Identity, TaskRef, OperationID, Now}`. | Забирает награду по готовой задаче и возвращает новый статус. |

## internal

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `Internal.OnPartnerCallback(ctx, params)` | `PartnerCallbackParams{WorkspaceID, Provider, IssueID, IssueRef, ExternalClickID, Status, Payload, Now}`. | Обрабатывает системный callback/postback партнера. Поддерживает completion по `external_click_id` для click-based партнеров и отмену (`revoked`, `unsubscribed`, `cancelled`): до claim блокирует выдачу награды статусом `revoked`, после claim ставит `revoked_after_claim`, пишет статистику и создает callback `task.partner.revoked` для внешней компенсации. |
| `Internal.HandlePartnerWebhook(ctx, params)` | `PartnerWebhookParams{WorkspaceID, Secret, Headers, Query, Body, Now}`. | Единая точка для webhook `/webhook/{workspace_id}/{secret}/`: по `Secret` находит partner config внутри workspace, передает request в Lua runtime партнера, нормализует результат и вызывает `OnPartnerCallback`. Позволяет подключать разные webhook/postback форматы без отдельного Go-контроллера на каждого партнера. |

## partner lua contract

Lua provider экспортирует не общий `handle`, а конкретные функции:

| Функция | Когда вызывается | Что возвращает |
| --- | --- | --- |
| `list(event)` | Получение партнерских заданий. | `{ok=true, tasks=[{external_id, external_type, start_mode, public_payload, private_payload, expires_at}]}`. |
| `start(event)` | Старт/click-generation задания, если партнеру нужен отдельный action URL. | `{ok=true, started=true, status, action_url, external_click_id, public_payload_patch, private_payload_patch}`. |
| `check(event)` | Проверка выполнения пользователем. | `{ok=true, completed=true/false, status, payload}`. |
| `callback(event)` | Входящий webhook/postback партнера. | `{ok=true, action="complete"|"revoked", status, issue_id, issue_ref, external_click_id, external_id, platform_user_id, payload}`. |

`event` содержит `action`, `provider`, `identity`, `config`, `issue`, `request`, `variables`, `locale`, `limit`, `now`. Runtime вызывает только функцию, совпадающую с action; если функции нет, вызов завершается ошибкой `has no {action}(event)`.

## admin

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `Admin.UpsertGroup(ctx, workspaceID, key, position, active)` | `workspaceID`, `key`, `position`, `active`. | Создает или обновляет группу задач. |
| `Admin.UpsertGroupLocalization(ctx, workspaceID, key, locale, title, description)` | Данные локализации группы. | Создает или обновляет локализацию группы. |
| `Admin.UpsertSequence(ctx, workspaceID, key, position, active)` | `workspaceID`, `key`, `position`, `active`. | Создает или обновляет последовательность задач. |
| `Admin.SaveTask(ctx, params)` | `SaveTaskParams{ID, WorkspaceID, Key, GroupKey, SequenceKey, SequencePosition, TaskKind, ActionKey, ActionKind, ClaimMode, StartMode, TargetCount, ResetUnit, ResetEvery, Position, Payload, Target, IntegrationKind, IntegrationProvider, IntegrationPayload, ImageURL, IsVisible, IsActive, StartAt, EndAt}`. | Создает или обновляет задачу, включая `start_mode=none|required`, target-фильтр отображения и закрытый integration payload. |
| `Admin.DeleteTask(ctx, workspaceID, id)` | `workspaceID`, `id`. | Удаляет задачу. |
| `Admin.GetTask(ctx, workspaceID, id)` | `workspaceID`, `id`. | Возвращает задачу. |
| `Admin.ListTasks(ctx, workspaceID, groupKey, limit, offset)` | `workspaceID`, `groupKey`, `limit`, `offset`. | Возвращает список задач, опционально по группе. |
| `Admin.UpsertTaskLocalization(ctx, workspaceID, taskID, locale, title, description)` | Данные локализации задачи. | Создает или обновляет локализацию задачи. |
| `Admin.UpsertReward(ctx, workspaceID, taskID, reward, position)` | `RewardModel{Key, Type, Quantity, Scale, Unit}`, `position`. | Создает или обновляет награду задачи. `Scale` задает точность дробной валюты, например `25/scale=2` = `0.25`. |
| `Admin.DeleteReward(ctx, workspaceID, taskID, key)` | `workspaceID`, `taskID`, `key`. | Удаляет награду задачи. |
| `Admin.ExportManifest(ctx)` | Только `ctx`. | Возвращает manifest доступных секций export/import для tasks. |
| `Admin.Export(ctx, workspaceID, req)` | `workspaceID`, `ExportRequest{Sections, GroupKeys, IncludeSecrets, Now}`. | Экспортирует задачи в `tasks.export.v1`: группы, последовательности, задачи, локализации, награды, target, интеграционные настройки, партнерские настройки и правила наград согласно выбранным секциям. |
| `Admin.PreviewImport(ctx, workspaceID, pkg)` | `workspaceID`, `ExportPackage`. | Проверяет пакет импорта, считает элементы и возвращает конфликты без записи данных. |
| `Admin.Import(ctx, workspaceID, req)` | `ImportRequest{Package, ConflictStrategy}`; стратегии `fail_on_conflict`, `skip_existing`, `update_existing`. | Импортирует выбранные секции задач пачками в транзакции и обновляет связи групп, задач, локализаций, наград, интеграций и партнеров. |
| `Admin.SavePartnerConfig(ctx, params)` | `PartnerConfigModel{WorkspaceID, Provider, GroupKey, Platform, IsEnabled, Secret, WebhookSecret, Target, Settings}`. | Создает или обновляет настройки партнера, включая API-секрет партнера, секрет входящего webhook и target. |
| `Admin.GetPartnerConfig(ctx, workspaceID, provider, groupKey, platform)` | Ключи конфигурации партнера. | Возвращает конфигурацию партнера. |
| `Admin.ListPartnerConfigs(ctx, workspaceID)` | `workspaceID`. | Возвращает все конфигурации партнеров workspace. |
| `Admin.SavePartnerScript(ctx, params)` | `PartnerScriptModel{Provider, IsEnabled, Version, Source}`. | Создает или обновляет Lua runtime provider. При изменении bump-ает версию кеша скриптов; runtime после обновления версии закрывает старый pool Lua states и поднимает новый. |
| `Admin.GetPartnerScript(ctx, provider)` | `provider`. | Возвращает Lua script provider-а. |
| `Admin.ListPartnerScripts(ctx)` | Только `ctx`. | Возвращает все Lua runtime provider-ы. |
| `Admin.SavePartnerRewardRule(ctx, params)` | `SavePartnerRewardRuleParams{WorkspaceID, Provider, GroupKey, ExternalType, Reward, Position, IsEnabled}`. | Создает или обновляет правило награды партнера; `ExternalType="*"` используется как дефолт. |
| `Admin.DeletePartnerRewardRule(ctx, workspaceID, provider, groupKey, externalType, rewardKey)` | Ключи правила награды. | Удаляет правило награды партнера. |
| `Admin.ListPartnerDailyStats(ctx, workspaceID, provider, groupKey, from, until)` | `workspaceID`, опциональные `provider/groupKey`, период. | Возвращает дневную статистику партнерских заданий по partner/group/type с уже инкрементально подготовленными счетчиками. |
| `Admin.GetStats(ctx, workspaceID)` | `workspaceID`. | Возвращает агрегированную статистику задач. |
| `Admin.GetTaskStats(ctx, workspaceID, taskID)` | `workspaceID`, `taskID`. | Возвращает статистику одной задачи. |
| `Admin.ListDailyStats(ctx, workspaceID, taskID, from, until)` | `workspaceID`, `taskID`, `from`, `until`. | Возвращает дневную статистику задачи. |
| `Admin.ListDailyOverview(ctx, workspaceID, from, until)` | `workspaceID`, `from`, `until`. | Возвращает дневный обзор по задачам. |
| `Admin.RefreshDailyStats(ctx, from, until)` | `from`, `until`. | Пересчитывает дневную статистику. |
