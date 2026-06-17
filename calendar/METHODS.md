# Calendar methods

Только методы слоев `user` и `admin`, которые можно использовать как основу будущего API.

## user

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `User.ListActive(ctx, workspaceID, locale, now)` | `workspaceID`, `locale`, `now time.Time`. | Возвращает активные календари рабочей области на момент `now`. |
| `User.GetCalendar(ctx, identity, ref, locale)` | `Identity{WorkspaceID, AppID, PlatformID, PlatformUserID}`, `ref`, `locale`. | Возвращает календарь с локализацией, шагами и наградами для пользователя. |
| `User.GetProgress(ctx, identity, calendarID)` | `Identity`, `calendarID`. | Возвращает прогресс пользователя по календарю. |
| `User.Record(ctx, params)` | `RecordParams{Identity, CalendarID, OperationID, Now}`. | Фиксирует операцию пользователя, обновляет прогресс и возвращает результат выдачи награды. |
| `User.Next(ctx, params)` | `NextParams{Identity, CalendarID, OperationID, Now}`. | Записывает переход пользователя к следующему доступному шагу календаря. |

## admin

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `Admin.CreateCalendar(ctx, params)` | `SaveCalendarParams{ID, WorkspaceID, Type, Mode, IntervalType, IntervalUnit, IntervalCount, ResetAfterIntervals, EndBehavior, Timezone, HideFutureRewards, IsActive, StartAt, EndAt}`. | Создает календарь; при пустом `ID` генерирует UUID. |
| `Admin.UpdateCalendar(ctx, params)` | `SaveCalendarParams` с обязательным `ID`. | Обновляет календарь. |
| `Admin.GetCalendar(ctx, workspaceID, id)` | `workspaceID`, `id`. | Возвращает календарь с локализациями, шагами и наградами. |
| `Admin.ListCalendars(ctx, workspaceID, page)` | `workspaceID`, `Page{Limit, Offset}`. | Возвращает список календарей. |
| `Admin.SetCalendarActive(ctx, workspaceID, id, active)` | `workspaceID`, `id`, `active bool`. | Включает или выключает календарь. |
| `Admin.DeleteCalendar(ctx, workspaceID, id)` | `workspaceID`, `id`. | Удаляет календарь. |
| `Admin.CreateStep(ctx, params)` | `SaveStepParams{WorkspaceID, CalendarID, Position}`. | Создает шаг календаря. |
| `Admin.UpdateStep(ctx, params)` | `SaveStepParams{WorkspaceID, CalendarID, ID, Position}`. | Обновляет шаг календаря. |
| `Admin.DeleteStep(ctx, workspaceID, calendarID, id)` | `workspaceID`, `calendarID`, `id`. | Удаляет шаг календаря. |
| `Admin.CreateReward(ctx, params)` | `SaveRewardParams{WorkspaceID, CalendarID, StepID, Key, Type, Quantity, Unit, Position}`. | Создает награду шага. |
| `Admin.UpdateReward(ctx, params)` | `SaveRewardParams` с обязательным `ID`. | Обновляет награду шага. |
| `Admin.GetReward(ctx, workspaceID, calendarID, id)` | `workspaceID`, `calendarID`, `id`. | Возвращает награду. |
| `Admin.DeleteReward(ctx, workspaceID, calendarID, id)` | `workspaceID`, `calendarID`, `id`. | Удаляет награду. |
| `Admin.UpsertLocalization(ctx, params)` | `SaveLocalizationParams{WorkspaceID, CalendarID, Locale, Title, Description}`. | Создает или обновляет локализацию календаря. |
| `Admin.GetLocalization(ctx, workspaceID, calendarID, locale)` | `workspaceID`, `calendarID`, `locale`. | Возвращает локализацию. |
| `Admin.ListLocalizations(ctx, workspaceID, calendarID)` | `workspaceID`, `calendarID`. | Возвращает локализации календаря. |
| `Admin.DeleteLocalization(ctx, workspaceID, calendarID, locale)` | `workspaceID`, `calendarID`, `locale`. | Удаляет локализацию. |
| `Admin.ListOperations(ctx, workspaceID, calendarID, page)` | `workspaceID`, `calendarID`, `Page`. | Возвращает журнал операций календаря. |
| `Admin.GetStats(ctx, workspaceID, calendarID)` | `workspaceID`, `calendarID`. | Возвращает агрегированную статистику календаря. |
| `Admin.ListDailyStats(ctx, workspaceID, calendarID, from, until)` | `workspaceID`, `calendarID`, `from`, `until`. | Возвращает дневную статистику за период. |
| `Admin.RefreshDailyStats(ctx, from, until)` | `from`, `until`. | Пересчитывает дневную статистику. |
| `Admin.ListCallbackEvents(ctx, params)` | `CallbackEventListParams{Status, Page}`. | Возвращает callback-события календаря. |
| `Admin.GetCallbackEvent(ctx, id)` | `id`. | Возвращает callback-событие. |
| `Admin.RetryCallbackEventNow(ctx, id)` | `id`. | Отправляет callback-событие на повторную обработку. |
| `Admin.MarkCallbackEventOK(ctx, id)` | `id`. | Помечает callback-событие успешным. |
| `Admin.MarkCallbackEventReject(ctx, id, reason)` | `id`, `reason`. | Помечает callback-событие отклоненным. |
| `Admin.ResetExpiredCallbackProcessing(ctx)` | Только `ctx`. | Сбрасывает зависшие callback-события в обработке. |

