# CPA methods

Только методы слоев `user` и `admin`, которые можно использовать как основу будущего API.

## user

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `User.ListActive(ctx, params)` | `ListActiveParams{Identity, Locale, Now}`; `Identity{WorkspaceID, AppID, PlatformID, Platform, PlatformUserID, IsPremium, Sex, Country}`. | Возвращает активные CPA-офферы для пользователя. |
| `User.GetCode(ctx, params)` | `GetCodeParams{Identity, CPAID}`. | Выдает или возвращает уже назначенный CPA-код пользователя. |
| `User.GetStatus(ctx, params)` | `GetStatusParams{Identity, CPAID}`. | Возвращает текущее назначение пользователя по CPA-офферу. |

## admin

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `Admin.UpsertOffer(ctx, params)` | `UpsertOfferParams{WorkspaceID, ID, Payload, CodeMode, CodeSource, SharedCode, GeneratedLength, GeneratedAlphabet, IsActive, StartAt, EndAt}`. | Создает или обновляет CPA-оффер и правила выдачи кодов. |
| `Admin.GetOffer(ctx, workspaceID, cpaID)` | `workspaceID`, `cpaID`. | Возвращает оффер с локализациями и наградами. |
| `Admin.ListOffers(ctx, workspaceID, page)` | `workspaceID`, `Page{Limit, Offset}`. | Возвращает список офферов. |
| `Admin.DeleteOffer(ctx, workspaceID, cpaID)` | `workspaceID`, `cpaID`. | Удаляет оффер. |
| `Admin.UpsertLocalization(ctx, params)` | `UpsertLocalizationParams{WorkspaceID, CPAID, Locale, Title, Description}`. | Создает или обновляет локализацию оффера. |
| `Admin.ListLocalizations(ctx, workspaceID, cpaID)` | `workspaceID`, `cpaID`. | Возвращает локализации оффера. |
| `Admin.DeleteLocalization(ctx, workspaceID, cpaID, locale)` | `workspaceID`, `cpaID`, `locale`. | Удаляет локализацию. |
| `Admin.UpsertReward(ctx, params)` | `UpsertRewardParams{WorkspaceID, CPAID, Key, Type, Quantity, Scale, Unit}`. | Создает или обновляет награду оффера. `Scale` задает точность дробной валюты, например `25/scale=2` = `0.25`. |
| `Admin.ListRewards(ctx, workspaceID, cpaID)` | `workspaceID`, `cpaID`. | Возвращает награды оффера. |
| `Admin.DeleteReward(ctx, workspaceID, cpaID, rewardID)` | `workspaceID`, `cpaID`, `rewardID`. | Удаляет награду оффера. |
| `Admin.Export(ctx, workspaceID, req)` | `workspaceID`, `ExportRequest{Now}`. | Экспортирует CPA-офферы workspace в `cpa.export.v1`: payload, target, локализации, награды и настройки кодов. |
| `Admin.PreviewImport(ctx, workspaceID, pkg)` | `workspaceID`, `ExportPackage`. | Проверяет пакет импорта, считает элементы и возвращает конфликты по `offer.ID` без записи данных. |
| `Admin.Import(ctx, workspaceID, req)` | `ImportRequest{Package, ConflictStrategy}`; стратегии `fail_on_conflict`, `skip_existing`, `update_existing`. | Импортирует CPA-офферы в workspace пачками в транзакции и сбрасывает кеш CPA. |
| `Admin.AddCodes(ctx, params)` | `AddCodesParams{WorkspaceID, CPAID, Codes}`. | Добавляет пул персональных кодов для оффера. |
| `Admin.DeleteAvailableCodes(ctx, workspaceID, cpaID)` | `workspaceID`, `cpaID`. | Удаляет доступные, еще не выданные коды оффера. |
| `Admin.DeleteIssuedCodes(ctx, workspaceID, cpaID)` | `workspaceID`, `cpaID`. | Удаляет выданные коды оффера. |
| `Admin.DeleteCompletedCodes(ctx, workspaceID, cpaID)` | `workspaceID`, `cpaID`. | Удаляет завершенные коды оффера. |
| `Admin.GetUserAssignment(ctx, params)` | `user.GetStatusParams{Identity, CPAID}`. | Возвращает assignment конкретного пользователя. |
| `Admin.ListAssignments(ctx, params)` | `AuditListParams{WorkspaceID, CPAID, Status, Page}`. | Возвращает список assignments. |
| `Admin.ListCodes(ctx, params)` | `AuditListParams{WorkspaceID, CPAID, Status, Page}`. | Возвращает список кодов оффера. |
| `Admin.ListAssignmentEvents(ctx, params)` | `AuditListParams{WorkspaceID, CPAID, Status, Page}`. | Возвращает события аудита assignments. |
| `Admin.Complete(ctx, params)` | `CompleteParams{Identity, CPAID}`. | Завершает assignment пользователя и возвращает выданные награды. |
| `Admin.GetStats(ctx, workspaceID, cpaID)` | `workspaceID`, `cpaID`. | Возвращает агрегированную статистику оффера. |
| `Admin.ListDailyStats(ctx, workspaceID, cpaID, from, until)` | `workspaceID`, `cpaID`, `from`, `until`. | Возвращает дневную статистику оффера. |
| `Admin.RefreshDailyStats(ctx, from, until)` | `from`, `until`. | Пересчитывает дневную статистику. |
| `Admin.ListCallbackEvents(ctx, params)` | `CallbackEventListParams{Status, Page}`. | Возвращает callback-события CPA. |
| `Admin.GetCallbackEvent(ctx, id)` | `id`. | Возвращает callback-событие. |
| `Admin.RetryCallbackEventNow(ctx, id)` | `id`. | Отправляет callback-событие на повторную обработку. |
| `Admin.MarkCallbackEventOK(ctx, id)` | `id`. | Помечает callback-событие успешным. |
| `Admin.MarkCallbackEventReject(ctx, id, reason)` | `id`, `reason`. | Помечает callback-событие отклоненным. |
| `Admin.ResetExpiredCallbackProcessing(ctx)` | Только `ctx`. | Сбрасывает зависшие callback-события в обработке. |
