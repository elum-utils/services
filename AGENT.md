# Services Style Guide

Обязательные правила проектирования и поддержки сервисов в этом репозитории.

## 1. Основные принципы

- Каждый сервис полностью независим:
  - собственная схема и repository;
  - собственные domain-модели;
  - собственный callback producer;
  - никаких связей таблиц или прямых вызовов другого бизнес-сервиса.
- Общими могут быть только технические utilities из `internal/utils`.
- Публичное API должно скрывать SQL, sqlc, bootstrap и устройство callback-очереди.
- Бизнес-инварианты должны защищаться не только Go-кодом, но и ограничениями БД:
  `UNIQUE`, `FOREIGN KEY`, `CHECK`, транзакциями и блокировками.
- Для горячих методов количество SQL round-trip является частью дизайна, а не
  задачей последующей оптимизации.

## 2. Структура сервиса

Корневой каталог самостоятельного сервиса:

```text
<service>/
  <service>.go
  config.go
  callback.go
  repository/
  service/
    admin/
    user/
  sqlc/
    schema.sql
    query.sql
    event.sql
    trigger.sql
  tests/
```

Необязательные файлы (`event.sql`, `trigger.sql`, `callback.go`) добавляются
только когда сервису действительно нужна соответствующая функциональность.

Корневая структура должна называться предметно:

```go
type Payment struct {}
type CPA struct {}
type Promo struct {}
```

Не использовать `Api`, `Manager`, `Service` как имя основной структуры.

Если у сервиса есть административное и пользовательское API:

```go
type Promo struct {
    Admin *admin.Admin
    User  *user.User
}
```

## 3. Слои

### `sqlc`

- Содержит SQL schema/query/event/trigger и сгенерированный код.
- Не содержит бизнес-логики на Go.
- Не используется напрямую из `service`.
- Сгенерированные файлы нельзя редактировать вручную.
- После изменения SQL обязательно запускать `sqlc generate`.

### `repository`

- Единственный слой, который работает с:
  - `sqlc`;
  - `database/sql`;
  - `sql.Null*`;
  - транзакциями;
  - DB-locking;
  - DB-oriented инвариантами.
- Принимает и возвращает обычные Go-типы и repository domain-модели.
- Сам выполняет преобразования `sql.Null*`.
- Собирает плоские строки JOIN-запросов в готовые domain-модели.
- Не возвращает `sqlc.*Row`, `sqlc.*Params` или `sql.Null*` наружу.

### `service`

- Содержит бизнес-валидацию и orchestration.
- Не импортирует `database/sql` и пакет `sqlc`.
- Не знает о колонках, nullable SQL-типах и механике запросов.
- Преобразует repository-модели в публичные модели API.
- Не дублирует DB-инварианты сложной логикой, если они уже атомарно защищены
  repository/БД.

### Корневой пакет

- Собирает `Admin`, `User`, adapters и callback store.
- Управляет root context и graceful shutdown.
- `New(DatabaseParams)` только создает и настраивает экземпляр, не открывая БД.
- `OnCallback` до запуска только регистрирует callback worker.
- `Run(ctx)` открывает БД, собирает внутренние сервисы, запускает workers и
  блокируется до отмены `ctx`, вызова `Close` или ошибки worker.
- `Run` самостоятельно выполняет graceful shutdown и закрывает принадлежащие
  сервису соединения.
- Для тестов и специального встраивания допускается
  `NewWithDatabase(ctx, *sql.DB, Options)`, который не владеет переданной БД.
- Не предоставлять `Open`: запуск и владение ресурсами должны быть скрыты под
  блокирующим `Run`.
- Публичный config не должен выставлять `sqlwrap.Options` напрямую.
- Публичные сигнатуры не должны содержать типы из `internal`.

## 4. Компоновка файлов

Пример `service/product`:

```text
product.go
models.go
create.go
get.go
list.go
delete_price.go
helpers.go
```

- `<service>.go`:
  - основная структура;
  - `New`;
  - `Close`;
  - `withContext`.
- `models.go`:
  - общие публичные модели;
  - типы, используемые несколькими методами.
- `<method>.go`:
  - один публичный метод или тесно связанная группа CRUD-методов;
  - params и result, используемые только этим методом.
- `helpers.go`:
  - общие внутренние mapper/helper;
  - не создавать ради одной тривиальной функции.

Не размещать в одном файле структуру сервиса, все DTO и большое количество
несвязанных методов.

## 5. Именование

- Сервис: `product.Product`, `admin.Admin`, `user.User`.
- Публичная модель при конфликте с сервисом:
  - `product.ProductModel`;
  - `PromoModel`;
  - `RewardModel`.
- Команда должна называться действием:
  - `Refund.Execute`, а не `Refund.Refund`.
- Административные методы находятся под `service.Admin`, а не смешиваются с
  пользовательскими.
- SQL-запросы получают предметные имена:
  - `GetApplyBundleForUpdate`;
  - `AdminListPromos`;
  - `CreateRedemption`.

## 6. Типы и преобразования

В `service` запрещены:

- `database/sql`;
- `sql.Null*`;
- `sqlc.*`;
- DB-specific enum types.

В публичных моделях использовать:

- обычные scalar-типы;
- указатели для nullable значений;
- `time.Time` / `*time.Time`;
- `json.RawMessage` для произвольного JSON payload;
- slices и domain-структуры.

Общие pointer helpers брать из `internal/utils`, не создавать локальные
`stringPtr`, `int64Ptr`, `deref` и их аналоги.

SQL-преобразования должны находиться максимально близко к repository.

## 7. Workspace-изоляция

- Все пользовательские и административные операции scoped по `workspace_id`.
- Один числовой `id` без `workspace_id` недостаточен для repository-метода.
- Бизнес-уникальность должна включать workspace:

```sql
UNIQUE KEY (... workspace_id, ...)
```

- Дочерние таблицы должны ссылаться на родителя составным FK:

```sql
FOREIGN KEY (workspace_id, entity_id)
REFERENCES parent (workspace_id, id)
```

- Родитель должен иметь подходящий `UNIQUE (workspace_id, id)`.
- Нельзя полагаться только на `WHERE workspace_id = ?` в Go-коде: БД также
  должна запрещать cross-workspace связи.

## 8. SQL и производительность

### Query budget

Для пользовательского горячего метода ориентир:

- read/idempotent path: 1 SQL round-trip;
- successful write path: не более 2 SQL round-trip;
- превышение допускается только с измеримым обоснованием.

Количество запросов должно быть понятно из repository-кода.

### Чтение агрегатов

- Не делать последовательные запросы `entity -> localization -> rewards ->
  state`.
- Использовать один плоский `JOIN`-запрос и собирать строки в Go.
- Не использовать `JSON_ARRAYAGG`/`JSON_OBJECTAGG` для обычных вложенных
  коллекций, если можно вернуть набор строк:
  это добавляет сериализацию в MySQL и десериализацию/аллокации в Go.
- Не использовать `SELECT *` в горячих JOIN-запросах. Перечислять только
  необходимые поля.
- Не допускать N+1.

### Индексы

- Каждый lookup/locking query должен иметь индекс, начинающийся с полей
  equality-фильтра.
- Индексы проектируются под реальные `WHERE`, `JOIN`, `ORDER BY`.
- После изменения горячего запроса проверять `EXPLAIN ANALYZE`.
- Избыточные индексы не добавлять: каждый индекс удорожает запись.

### Денормализация

Денормализация допустима и желательна, когда она:

- уменьшает round-trip горячего метода;
- сохраняет исторически выданные данные;
- устраняет гонку между записью и callback payload.

Пример: snapshot награды в записи выдачи. Изменение текущей награды после
выдачи не должно менять уже выданную награду или callback.

Snapshot должен храниться в структурированном JSON только если набор данных
естественно является неизменяемым документом. Основные searchable поля
остаются отдельными колонками.

## 9. Транзакции и конкурентность

- Операции `check -> write -> outbox` выполняются атомарно.
- Для lifetime/global limit использовать блокировку общей строки:
  `SELECT ... FOR UPDATE`.
- Идемпотентность защищается `UNIQUE`, а не только предварительным `SELECT`.
- Повторный вызов не должен повторно:
  - выдавать награду;
  - увеличивать счетчик;
  - создавать callback.
- Нельзя открывать транзакцию в `service`; транзакциями владеет repository.
- Rollback должен отменять business write, raw event и callback outbox вместе.
- DB-trigger допустим для коротких атомарных действий, непосредственно
  являющихся следствием вставки:
  - счетчик;
  - raw event;
  - outbox event.
- Trigger не должен содержать сетевые вызовы, сложные циклы или скрытую
  бизнес-оркестрацию.
- Trigger SQL хранится отдельно и устанавливается bootstrap-ом.

## 10. Callback/outbox

- Callback создается в той же транзакции, что и бизнес-событие.
- Нельзя сначала commit бизнес-запись, а затем отдельной транзакцией создавать
  callback.
- `event_key` обязан быть детерминированным и уникальным.
- Повторное применение/выдача не создает второй callback.
- Payload содержит snapshot фактически выданных данных.
- `OnCallback` скрывает leasing, retry, success/fail/reject и десериализацию.
- Сервис десериализует payload в предметный `Context`, а не отдает наружу
  сырые байты очереди.
- Админ API callback-системы должен позволять:
  - получить/list события;
  - повторить сейчас;
  - отметить `ok`;
  - отметить `reject`;
  - сбросить истекший processing lease.

## 11. Bootstrap

- Пользователь сервиса не должен передавать путь к `schema.sql`.
- SQL-файлы встраиваются через `go:embed` в `sqlc/bootstrap.go`.
- `repository.Bootstrap(ctx)` устанавливает:
  1. schema;
  2. общую callback schema;
  3. triggers;
  4. DB events.
- Bootstrap должен быть идемпотентным.
- Payment не менять автоматически при приведении новых сервисов к этому
  правилу: отдельный существующий сервис меняется только отдельной задачей.

## 12. Контексты и shutdown

- `New` принимает root lifecycle context.
- Каждый публичный метод принимает request context.
- Метод выполняется на merged context:
  - отмена root context останавливает все операции сервиса;
  - отмена request context останавливает конкретный запрос.
- При `nil` context использовать нормализованный `context.Background()`.
- `Close`:
  1. отменяет root context;
  2. ждет background workers;
  3. закрывает сервисы/callback store;
  4. закрывает DB client только если сервис владеет им.

## 13. Admin и User API

- `User` содержит только минимальные пользовательские сценарии.
- `Admin` содержит полный CRUD и операционные методы.
- Soft-deleted сущности нельзя физически удалять, если нужна история.
- Admin CRUD обязан охватывать:
  - create/update/get/list/delete или soft delete;
  - localization CRUD;
  - reward CRUD;
  - raw records;
  - aggregated statistics;
  - lookup статуса конкретного пользователя;
  - callback administration, если сервис создает callbacks.

## 14. Статистика

- Raw события сохраняются отдельно и не изменяются.
- Суммарная ежедневная статистика хранится в отдельной таблице.
- Для ежедневной агрегации использовать MySQL Event, когда не требуется Go
  orchestration.
- Стандартное расписание:

```sql
EVERY '1' DAY
STARTS '2025-11-08 00:05:00'
```

- Должен существовать admin-метод ручного пересчета диапазона.
- `unique_users` считает business identity, а не количество строк событий.

## 15. Тесты

Минимальный интеграционный набор нового сервиса:

- полный successful lifecycle;
- idempotent repeat;
- все публичные status/error outcomes;
- workspace isolation;
- case normalization, если применимо;
- soft delete;
- global/lifetime limit;
- конкурентный тест лимита;
- callback создается ровно один раз;
- callback payload полностью десериализуется;
- snapshot не меняется после редактирования исходных данных;
- admin CRUD;
- raw и daily statistics;
- graceful context cancellation.

Тесты должны работать с реальной MySQL-схемой сервиса, когда проверяются SQL,
транзакции, locks, triggers или events.

## 16. Бенчмарки

- Для публичных горячих методов обязательны service benchmarks.
- Всегда включать `-benchmem`.
- Отдельно измерять:
  - successful path;
  - idempotent/read path;
  - ключевые admin reads.
- Пользователи/ключи successful benchmark должны быть уникальны между
  калибровочными запусками `testing.B`.
- После SQL-оптимизации фиксировать до/после:
  - `ns/op`;
  - `B/op`;
  - `allocs/op`;
  - количество SQL round-trip.
- Оптимизация не считается завершенной без повторного конкурентного теста.

## 17. Порядок создания нового сервиса

1. Зафиксировать domain-инварианты, identity и workspace scope.
2. Спроектировать таблицы, уникальности, FK и индексы.
3. Спроектировать горячие методы и их query budget.
4. Добавить `schema.sql`, `query.sql`, при необходимости `trigger.sql` и
   `event.sql`.
5. Запустить `sqlc generate`.
6. Создать repository и спрятать все DB-типы.
7. Создать `service/admin` и `service/user`.
8. Создать корневой wiring, config и callback wrapper.
9. Встроить SQL и реализовать скрытый bootstrap.
10. Добавить интеграционные тесты, concurrency tests и benchmarks.
11. Запустить:

```bash
sqlc generate
gofmt -w <service>
go vet ./<service>/...
go test -count=1 ./<service>/...
go test -run '^$' -bench . -benchmem ./<service>/tests
```

12. Для горячих запросов проверить `EXPLAIN ANALYZE`.

## 18. Правила рефакторинга

1. Не менять публичное поведение без отдельного требования.
2. Сначала отделить компоновку файлов от логики.
3. Затем убрать DB-типы из service.
4. Затем оптимизировать SQL и количество round-trip.
5. Не смешивать архитектурный рефакторинг с несвязанными изменениями.
6. Не редактировать generated sqlc вручную.
7. После каждого этапа запускать целевые тесты.
8. В конце запускать `go vet`, полный набор тестов сервиса и benchmarks.
