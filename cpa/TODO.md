# TODO: CPA

Список составлен по подтверждённым результатам аудита от 2026-07-11.

## P1

- [x] Сделать `fail_on_conflict` и `skip_existing` атомарными относительно
  конкурентных admin write. Import берёт transaction-level workspace lock до
  preview и bulk insert; test подтверждает, что конкурентный оффер не
  перезаписывается.
  [import.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/import.go:74)

- [x] Валидировать весь вложенный import-пакет до транзакции. Localization и
  reward используют те же repository rules, что admin upsert; проверяются
  дубли `offer.id` и `reward.key`. Ошибка возвращает путь вида
  `offers[4].rewards[1].quantity`, до transaction нет записей.
  [import.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/import.go:260)

- [x] Исправить полный экспорт и поиск конфликтов. Export использует отдельный
  repository-путь без pagination, а PreviewImport получает все offer ID без
  кэша и лимита. Добавлен тест export и `fail_on_conflict` на 1001 оффере.
  [export.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/export.go:17)
  [import.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/import.go:215)

- [x] Зафиксировать семантику удаления code rows: assignment не удаляется и
  продолжает возвращаться пользователю, а `UNIQUE` сохраняет запрет на новую
  выдачу. `DeleteIssuedCodes` и `DeleteCompletedCodes` меняют только статус
  связанного `cpa_code` на `deleted`. Добавлен тест для issued и completed.
  [assignments.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/assignments.go:320)
  [query.sql](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/sqlc/query.sql:447)

- [x] Перевести кэшированные чтения CPA на version scope. Кэш разделен на
  области конкретного оффера, admin-списков и user-каталога; изменения
  инвалидают только затронутый оффер и два агрегированных списка. Удален
  process-local `cpaCacheKeys`. Добавлен тест двух CPA-нод с общим L2 и
  независимыми L1.
  [cache.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/cache.go:1)

- [x] Перенести CPA-тесты и benchmark-методы в два отслеживаемых файла:
  `cpa/cpa_test.go` и `cpa/cpa_bench_test.go`. Удалить старые файлы из
  игнорируемой папки `cpa/tests`.
  [cpa_test.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/cpa_test.go:1)

## P2

- [x] Убрать зависимость доступности оффера от TTL каталога. Каталог кэширует
  конфигурацию, но `start_at`/`end_at` фильтруются на каждом user request.
  [offers.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/offers.go:350)

- [x] Строго валидировать target rules до сохранения и import. Некорректные
  формы, неизвестные поля и пустые элементы списков отклоняются.
  [target.go](/Volumes/CLOUD/GitHub/elum-utils/services/internal/utils/target/target.go:66)

- [x] Читать CPA export в `REPEATABLE READ READ ONLY` snapshot, чтобы offer,
  localization и rewards не расходились при параллельном admin update.
  [export.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/export.go:16)

- [x] Проверять generation всех SQLC-конфигураций в CI: обход `sqlc.yaml`
  плюс чистый diff.
  [.github/workflows/ci.yml](/Volumes/CLOUD/GitHub/elum-utils/services/.github/workflows/ci.yml:1)

- [x] Разделить параметры admin audit. `ListAssignmentEvents` принимает
  `AssignmentEventListParams{WorkspaceID, CPAID, EventType, Page}`;
  assignments и codes используют отдельные typed params. Добавлен тест
  фильтрации completed event.
  [audit.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/service/admin/audit.go:111)

- [x] Убрать `items` из CPA import/export. CPA хранит только `reward.key` и
  не дублирует item-данные, которыми владеет Reference. Нет dependency
  manifest, валидации, import counts или result-полей для items.
  [export_models.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/export_models.go:19)

- [x] Валидировать каждый `ExportOffer` теми же правилами, что admin upsert,
  до открытия транзакции. `ImportValidationError` содержит индекс оффера и
  имя поля; пакет не записывается частично.
  [import.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/import.go:17)

- [x] Разбивать import bulk upsert на ограниченные пачки. Лимит — 1000 строк
  и не более 60 000 параметров на SQL-команду. Тест импортирует 5500 офферов:
  единый `cpa_offer` запрос содержал бы 66 000 параметров.
  [import.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/import.go:14)

- [x] Ввести общую проверку `Identity` для public user-операций: непустые
  workspace/platform user ID и положительные app/platform IDs. Проверка
  выполняется в user-слое и дополнительно в repository перед assignment SQL.
  [models.go](/Volumes/CLOUD/GitHub/elum-utils/services/models.go:29)

- [x] Разделить успешный DB-write и ошибку инвалидации кэша. Выбрана
  диагностическая стратегия: write возвращает успех, а ошибка version bump
  передается в `Options.OnCacheInvalidationError`. Пока общий L2 недоступен,
  другая нода может видеть старую версию не дольше TTL.
  [cache.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/repository/cache.go:41)

- [x] Добавить тесты на: кэш между двумя нодами, delete/reissue,
  export/import >1000, большой import, invalid import, пустую identity и
  ошибку внешнего cache backend.
  [cpa_test.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/cpa_test.go:1)

## P3

- [x] Привести вручную поддерживаемый CPA-код к единому правилу: логические
  блоки отделяются пустой строкой; импорты разделены на standard/third-party/
  local; составные литералы и длинные вызовы записаны по одному полю или
  аргументу на строку. Generated `sqlc` не редактируется вручную.
  [get_code.go](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/service/user/get_code.go:16)

- [x] Обновить `METHODS.md`: убрать `Now` из `User.ListActive`, добавить
  `Target` в `UpsertOffer`, заменить `rewardID` на `rewardKey`.
  [METHODS.md](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/METHODS.md:9)

- [x] Удалить неиспользуемые query: `ListActiveOfferBundles`,
  `ListActiveOfferBundlesCTE`, `ListActiveOffers`,
  `ListLocalizationsForOffers`, `ListRewardsForOffers` и
  `ListAssignmentsForOffers`. User flow использует cached catalog и отдельное
  user assignment lookup.
  [query.sql](/Volumes/CLOUD/GitHub/elum-utils/services/cpa/sqlc/query.sql:87)

- [x] Заменить хрупкий SQL runner. `sqlwrap.SplitStatements` понимает quoted
  strings, line/nested block comments и PostgreSQL dollar-quoted тела; unit
  tests покрывают semicolon внутри функции и ошибку незакрытого синтаксиса.
  [statements.go](/Volumes/CLOUD/GitHub/elum-utils/services/internal/utils/sql/statements.go:10)
