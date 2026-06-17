# Reference methods

Только методы слоев `user` и `admin`, которые можно использовать как основу будущего API.

## user

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `User.Get(ctx, params)` | `GetParams{WorkspaceID, Key, Locale}`. | Возвращает активный справочный item по ключу. |
| `User.Resolve(ctx, params)` | `ResolveParams{WorkspaceID, Keys, Locale}`. | Массово разрешает ключи в items и возвращает найденные элементы плюс `MissingKeys`. |
| `User.List(ctx, workspaceID, locale, page)` | `workspaceID`, `locale`, `Page{Limit, Offset}`. | Возвращает страницу активных справочных items. |

## admin

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `Admin.CreateItem(ctx, params)` | `SaveItemParams{WorkspaceID, Key, Type, Payload, IsActive}`. | Создает справочный item. |
| `Admin.UpdateItem(ctx, params)` | `UpdateItemParams{WorkspaceID, Key, Payload, IsActive}`. | Обновляет payload и активность item. |
| `Admin.DangerousChangeType(ctx, params)` | `DangerousChangeTypeParams{WorkspaceID, Key, CurrentType, NewType, Confirmation}`. | Меняет тип item при подтверждении `CHANGE_REFERENCE_TYPE`. |
| `Admin.GetItem(ctx, workspaceID, key)` | `workspaceID`, `key`. | Возвращает item с локализациями и служебными полями. |
| `Admin.ListItems(ctx, params)` | `ItemListParams{WorkspaceID, Type, OnlyNotDeleted, Page}`. | Возвращает список items с фильтрами. |
| `Admin.SoftDeleteItem(ctx, workspaceID, key)` | `workspaceID`, `key`. | Мягко удаляет item. |
| `Admin.RestoreItem(ctx, workspaceID, key, active)` | `workspaceID`, `key`, `active bool`. | Восстанавливает item и задает активность. |
| `Admin.UpsertLocalization(ctx, params)` | `SaveLocalizationParams{WorkspaceID, ItemKey, Locale, Title, Description}`. | Создает или обновляет локализацию item. |
| `Admin.GetLocalization(ctx, workspaceID, key, locale)` | `workspaceID`, `key`, `locale`. | Возвращает локализацию. |
| `Admin.ListLocalizations(ctx, workspaceID, key)` | `workspaceID`, `key`. | Возвращает локализации item. |
| `Admin.DeleteLocalization(ctx, workspaceID, key, locale)` | `workspaceID`, `key`, `locale`. | Удаляет локализацию. |
| `Admin.GetStats(ctx, workspaceID)` | `workspaceID`. | Возвращает статистику справочника. |

