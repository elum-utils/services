# Tasks methods

Только методы слоев `user` и `admin`, которые можно использовать как основу будущего API.

## user

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `User.ListActive(ctx, identity, locale, now)` | `Identity{WorkspaceID, AppID, PlatformID, PlatformUserID}`, `locale`, `now`. | Возвращает активные задачи пользователя с прогрессом и локализацией. |
| `User.ListPartner(ctx, params)` | `PartnerListParams{Identity, Provider, GroupKey, Platform, Locale, Limit, Variables, Now}`. | Запрашивает партнерскую группу, скрыто вызывает provider adapter, создает/переиспользует `partner_issue` и возвращает партнерские задания в той же форме, что обычные задачи. |
| `User.CheckPartner(ctx, params)` | `PartnerCheckParams{Identity, IssueRef, Variables, Now}`. | Проверяет выполнение партнерского задания через provider adapter, скрывает партнерский API и при успехе помечает issue как `completed`, возвращая задачу со статусом `ready`. |
| `User.Claim(ctx, params)` | `ClaimParams{Identity, TaskRef, OperationID, Now}`. | Забирает награду по готовой задаче и возвращает новый статус. |

## admin

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `Admin.UpsertGroup(ctx, workspaceID, key, position, active)` | `workspaceID`, `key`, `position`, `active`. | Создает или обновляет группу задач. |
| `Admin.UpsertGroupLocalization(ctx, workspaceID, key, locale, title, description)` | Данные локализации группы. | Создает или обновляет локализацию группы. |
| `Admin.UpsertSequence(ctx, workspaceID, key, position, active)` | `workspaceID`, `key`, `position`, `active`. | Создает или обновляет последовательность задач. |
| `Admin.SaveTask(ctx, params)` | `SaveTaskParams{ID, WorkspaceID, Key, GroupKey, SequenceKey, SequencePosition, ActionKey, ActionKind, ClaimMode, TargetCount, ResetUnit, ResetEvery, Position, Payload, ImageURL, IsVisible, IsActive, StartAt, EndAt}`. | Создает или обновляет задачу. |
| `Admin.DeleteTask(ctx, workspaceID, id)` | `workspaceID`, `id`. | Удаляет задачу. |
| `Admin.GetTask(ctx, workspaceID, id)` | `workspaceID`, `id`. | Возвращает задачу. |
| `Admin.ListTasks(ctx, workspaceID, groupKey, limit, offset)` | `workspaceID`, `groupKey`, `limit`, `offset`. | Возвращает список задач, опционально по группе. |
| `Admin.UpsertTaskLocalization(ctx, workspaceID, taskID, locale, title, description)` | Данные локализации задачи. | Создает или обновляет локализацию задачи. |
| `Admin.UpsertReward(ctx, workspaceID, taskID, reward, position)` | `RewardModel{Key, Type, Quantity, Unit}`, `position`. | Создает или обновляет награду задачи. |
| `Admin.DeleteReward(ctx, workspaceID, taskID, key)` | `workspaceID`, `taskID`, `key`. | Удаляет награду задачи. |
| `Admin.SavePartnerConfig(ctx, params)` | `PartnerConfigModel{WorkspaceID, Provider, GroupKey, Platform, IsEnabled, Secret, Target, Settings}`. | Создает или обновляет настройки партнера, включая секреты и target. |
| `Admin.GetPartnerConfig(ctx, workspaceID, provider, groupKey, platform)` | Ключи конфигурации партнера. | Возвращает конфигурацию партнера. |
| `Admin.ListPartnerConfigs(ctx, workspaceID)` | `workspaceID`. | Возвращает все конфигурации партнеров workspace. |
| `Admin.SavePartnerRewardRule(ctx, params)` | `SavePartnerRewardRuleParams{WorkspaceID, Provider, GroupKey, ExternalType, Reward, Position, IsEnabled}`. | Создает или обновляет правило награды партнера; `ExternalType="*"` используется как дефолт. |
| `Admin.DeletePartnerRewardRule(ctx, workspaceID, provider, groupKey, externalType, rewardKey)` | Ключи правила награды. | Удаляет правило награды партнера. |
| `Admin.ListPartnerDailyStats(ctx, workspaceID, provider, groupKey, from, until)` | `workspaceID`, опциональные `provider/groupKey`, период. | Возвращает дневную статистику партнерских заданий по partner/group/type с уже инкрементально подготовленными счетчиками. |
| `Admin.GetStats(ctx, workspaceID)` | `workspaceID`. | Возвращает агрегированную статистику задач. |
| `Admin.GetTaskStats(ctx, workspaceID, taskID)` | `workspaceID`, `taskID`. | Возвращает статистику одной задачи. |
| `Admin.ListDailyStats(ctx, workspaceID, taskID, from, until)` | `workspaceID`, `taskID`, `from`, `until`. | Возвращает дневную статистику задачи. |
| `Admin.ListDailyOverview(ctx, workspaceID, from, until)` | `workspaceID`, `from`, `until`. | Возвращает дневный обзор по задачам. |
| `Admin.RefreshDailyStats(ctx, from, until)` | `from`, `until`. | Пересчитывает дневную статистику. |
