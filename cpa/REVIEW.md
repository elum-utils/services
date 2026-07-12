# Аудит CPA

Дата: 2026-07-11.

Проверены публичные слои `user` и `admin`, репозиторий, PostgreSQL-схема и
`sqlc`-запросы, кэш, импорт/экспорт, callback lifecycle и имеющиеся тесты.
Ниже отмечены только подтверждённые по коду проблемы.

## Оценка

| Направление | Оценка | Краткий вывод |
| --- | ---: | --- |
| Качество кода | 9/10 | Ручной код выровнен по блокам и импортам; статусы публичных моделей типизированы, а SQLC-код проверяется в CI. |
| Безопасность и надёжность | 9/10 | SQL параметризован, import защищён от конкурентных конфликтов, target конфигурация валидируется строго, выдача и completion используют транзакции. |
| Проектирование | 9/10 | Admin/user/repository, FK, versioned cache и snapshot export образуют последовательную основу. Audit API разделяет assignment/code status и event type. |

## Что сделано хорошо

- Public API разделён на `admin` и `user`, а SQL находится в `sqlc`-слое.
- Выдача кода и завершение assignment защищены транзакциями; для пулов кодов
  применяется `FOR UPDATE SKIP LOCKED`, а повторная выдача одному пользователю
  обработана идемпотентно.
- У схемы есть FK, `CHECK` и уникальные ограничения для конфигурации оффера,
  наград и кодов.
- Генерируемые промокоды используют криптографический источник случайности.
- CI проверяет форматирование, `go vet`, компиляцию, весь test suite и
  актуальность сгенерированного SQLC-кода.

### Исправлено 2026-07-11: import conflict strategy была уязвима к конкурентной записи

`Import` получает transaction-scoped advisory lock по `workspace_id` внутри
той же transaction, где строит preview и выполняет bulk write. `UpsertOffer`
и `DeleteOffer` используют тот же ключ, поэтому конфликт не может появиться
между проверкой и записью, но разные workspace не блокируют друг друга.
`fail_on_conflict` возвращает ошибку, а `skip_existing` сохраняет свою
семантику. Интеграционный тест удерживает competing lock, создаёт оффер в
конкурентной transaction и подтверждает, что import не перезаписывает его.

Ссылки: [import.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/import.go:74), [cpa_test.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/cpa_test.go:1032).

### Исправлено 2026-07-11: time window не должен устаревать вместе с каталогом

Кэшированный SQL-каталог содержит все `is_active` офферы и их конфигурацию.
`ListActiveForUser` на каждом запросе вычисляет `start_at` и `end_at` по
текущему UTC-времени. Поэтому оффер исчезает сразу после срока действия,
даже когда каталог взят из L1/L2 cache. Отдельный тест прогревает cache,
ждёт окончания оффера и подтверждает его отсутствие.

Ссылки: [offers.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/offers.go:350), [query.sql](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/sqlc/query.sql:87), [cpa_test.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/cpa_test.go:334).

### Исправлено 2026-07-11: target принимает только корректные правила

`target.Validate` требует объект с разрешёнными ключами; `is_premium` должен
быть bool, а списковые правила — непустыми строками/числами либо массивами
таких значений. `ValidateOffer` вызывает эту проверку и для admin upsert, и
для import preflight. Некорректный JSON больше не может превратиться в пустое
правило и открыть оффер всем пользователям.

Ссылки: [target.go](/Volumes/CLOUD/GitHub/elum-utils/services/internal/utils/target/target.go:66), [offers.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/offers.go:59), [target_test.go](/Volumes/CLOUD/GitHub/elum-utils/services/internal/utils/target/target_test.go:73).

### Исправлено 2026-07-11: export собирает консистентный снимок

`Export` читает offer, localization и rewards в `REPEATABLE READ READ ONLY`
transaction. Параллельная admin-запись теперь не может смешать разные версии
одного оффера в экспортируемом пакете.

Ссылки: [export.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/export.go:16).

### Исправлено 2026-07-11: CI проверяет SQLC generation

Workflow устанавливает закреплённую версию `sqlc`, находит и генерирует все
`sqlc.yaml` в репозитории, затем требует чистый `git diff`. Изменение
`query.sql` без обновления generated-кода любого сервиса теперь не пройдёт CI.

Ссылки: [ci.yml](/Volumes/CLOUD/GitHub/elum-utils/services/.github/workflows/ci.yml:1), [sqlc.yaml](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/sqlc.yaml:1).

### Исправлено 2026-07-11: статусы публичных контрактов имеют доменные типы

Введены `cpa.AssignmentStatus`, `cpa.CodeStatus` и
`cpa.AssignmentEventType`. Public models, callback payload и list-фильтры
используют эти типы; `AssignmentListParams` и `CodeListParams` разделены,
чтобы не смешивать разные enum. JSON остается строковым и не меняет API для
клиентов.

Ссылки: [types.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/types.go:1), [audit.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/service/admin/audit.go:10), [cpa_test.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/cpa_test.go:68).

## Подтверждённые проблемы

### Исправлено 2026-07-11: экспорт и проверка конфликтов теряли офферы после первой тысячи

`Export` теперь вызывает `ListAllOfferBundles`, который выполняет два
непагинируемых SQL-запроса. `PreviewImport` получает все ID через отдельный
`AdminListOfferIDs` без cache/pagination. Тест создаёт 1001 оффер, проверяет
полный export и конфликт на ранее пропускаемой 1001-й записи.

Ссылки: [export.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/export.go:17), [import.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/import.go:215), [offers.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/offers.go:653).

### Исправлено 2026-07-11: удалённый assignment скрывался, но блокировал новую выдачу кода

`DeleteIssuedCodes` и `DeleteCompletedCodes` теперь обновляют только
`cpa_code.status` до `deleted`. Assignment не скрывается от user-методов,
выданный код и завершенная награда остаются в истории, а абсолютный UNIQUE
сохраняет запрет на повторную выдачу. Тест покрывает оба статуса.

Ссылки: [query.sql](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/sqlc/query.sql:352), [query.sql](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/sqlc/query.sql:375), [query.sql](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/sqlc/query.sql:448), [schema.sql](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/sqlc/schema.sql:142).

### Исправлено 2026-07-11: кэш был версионирован только для пользовательского каталога

`GetOffer`, локализации и награды используют version scope конкретного
оффера; `ListOffers` и `ListOfferBundles` — scope admin-списков;
`ListActiveOfferCatalog` — scope user-каталога. Любое изменение делает старые
ключи недостижимыми без process-local реестра. Тест использует две CPA-ноды с
независимыми L1 и общим L2, затем проверяет обновление оффера, admin-списка и
user-каталога на второй ноде.

Ссылки: [cache.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/cache.go:9), [offers.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/offers.go:74), [offers.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/offers.go:242), [query.go](/Volumes/CLOUD/GitHub/elum-utils/services/internal/utils/sql/query.go:25).

### Исправлено 2026-07-11: тесты и бенчмарки не попадали в Git и CI

Корневой `.gitignore` игнорировал каждую папку с именем `tests`. Старые
файлы перенесены в два отслеживаемых корневых файла: `cpa/cpa_test.go` и
`cpa/cpa_bench_test.go`; исходная игнорируемая папка удалена. Публичные
сценарии теперь проверяются из чистого checkout.

Ссылки: [.gitignore](/Volumes/CLOUD/GitHub/elum-utils/services/.gitignore:17), [cpa_test.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/cpa_test.go:1), [cpa_bench_test.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/cpa_bench_test.go:1).

### Исправлено 2026-07-11: публичный импорт декларировал `items`, но их не импортировал

CPA больше не экспортирует и не импортирует `items`. Item-данными владеет
только сервис Reference, а CPA хранит ключ item в `reward.key` как ссылку без
валидации и без dependency manifest. Удалены поле пакета, import counts,
result-поля и сборщик items; тест проверяет отсутствие `items` в JSON export.

Ссылки: [export.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/export.go:9), [import.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/import.go:188), [export_models.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/export_models.go:19).

### Исправлено 2026-07-11: import валидируется до транзакции и разбивается на безопасные пачки

`ValidateOffer` — единый контракт для admin upsert и import. При ошибке
`ImportValidationError` содержит `offer_index` и `field`; до начала
транзакции не выполняется ни одной записи. Bulk upsert ограничен 1000 строками
и 60 000 параметрами. Тест импортирует 5500 офферов, для которых единый
`cpa_offer` запрос содержал бы 66 000 параметров.

Ссылки: [offers.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/offers.go:57), [import.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/import.go:14), [cpa_test.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/cpa_test.go:629).

### Исправлено 2026-07-11: public user-методы допускают только полную identity

Общий `Identity.Validate` требует непустые workspace и platform user ID,
положительные app/platform IDs. Проверка находится в каждом public user-методе
и продублирована в repository перед assignment-чтениями и записями, чтобы
прямой вызов repository не создал технический assignment.

Ссылки: [models.go](/Volumes/CLOUD/GitHub/elum-utils/services/models.go:29), [assignments.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/assignments.go:43), [cpa_test.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/cpa_test.go:173).

### Исправлено 2026-07-11: cache backend не меняет результат успешного DB-write

После commit изменение вызывает version bump. Ошибка L2-кэша не возвращается
клиенту как ошибка уже сохранённой операции; она синхронно передаётся в
`Options.OnCacheInvalidationError`. Это диагностическая стратегия: пока Redis
недоступен, другая нода может обслужить старую версию не дольше TTL, зато
клиент не повторяет уже выполненную запись. Тест эмулирует отказ version store
и подтверждает одновременно сохранение оффера и вызов diagnostic callback.

Ссылки: [cache.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/cache.go:41), [config.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/config.go:25), [cpa_test.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/cpa_test.go:666).

### Исправлено 2026-07-11: P3 code style, unused SQLC API и SQL splitter

Вручную поддерживаемые user/admin/repository методы разделены на логические
блоки: context, validation, repository action и mapping/return. Импорты
разделены на standard/third-party/local, а длинные аргументы и литералы
записываются с новой строки. Generated sqlc остаётся результатом генерации.

Удалены шесть неиспользуемых query и их prepared statements. User list
использует `ListActiveOfferCatalog` с versioned cache и отдельный assignment
lookup. Вместо `strings.Split` bootstrap использует `sqlwrap.SplitStatements`,
который корректно обрабатывает quoted strings, комментарии и PL/pgSQL тела.

Ссылки: [get_code.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/service/user/get_code.go:16), [query.sql](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/sqlc/query.sql:87), [statements.go](/Volumes/CLOUD/GitHub/elum-utils/services/internal/utils/sql/statements.go:10).

### Исправлено 2026-07-11: nested import проходит полный preflight до transaction

`ValidateLocalization` и `NormalizeAndValidateReward` теперь являются общими
правилами для admin upsert и import. Import дополнительно запрещает дубли
`offer.id` и `reward.key` в одном пакете. Ошибка возвращает `INVALID_FIELDS`
и путь `offers[index].localizations...` или `offers[index].rewards...`; тесты
проверяют отсутствие даже первой валидной записи при ошибке второй.

Ссылки: [import.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/import.go:260), [offers.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/offers.go:125), [cpa_test.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/cpa_test.go:629).

### Исправлено 2026-07-11: event filter отделён от status

`ListAssignmentEvents` принимает отдельный
`AssignmentEventListParams{WorkspaceID, CPAID, EventType, Page}`.
`AssignmentListParams.Status` и `CodeListParams.Status` используют разные
доменные enum. Новый тест создаёт issued и completed event и проверяет выборку
только completed.

Ссылки: [audit.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/service/admin/audit.go:17), [assignments.go](/Volumes/CLOUD/GitHub/el-utils/services/cpa/repository/assignments.go:132), [cpa_test.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/cpa_test.go:729).

## Не считаю проблемами

- Таргет фильтруется только при выдаче списка. Это соответствует ранее
  согласованному правилу: target регулирует отображение материала, а не
  возможность прямого системного действия.
- `Admin.Complete` доступен в административном слое, поэтому контроль
  полномочий ожидается в control/REST boundary, а не в CPA-пакете.
- Для generated code используется `crypto/rand`, поэтому предсказуемая
  генерация здесь не обнаружена.

## Проверка

В текущем окружении успешно выполнены:

```text
sqlc generate -f cpa/sqlc.yaml
test -z "$(gofmt -l .)"
go vet ./...
go build ./...
go test -count=1 ./...
```
