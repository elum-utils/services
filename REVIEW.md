# Повторный аудит сервисов

Актуально на 2026-07-11. Оценки учитывают уже внесённые исправления; открытые
пункты перечислены в `TODO.md` каждого сервиса.

| Сервис | Код | Безопасность | Проектирование | Основной остаточный риск |
|---|---:|---:|---:|---|
| CPA | 8/10 | 9/10 | 8/10 | Общие callback/stat contracts изменены после исходного CPA-аудита; нужен финальный regression pass. |
| Calendar | 7/10 | 8/10 | 8/10 | Нет двухнодового cache-теста и конкурентного import-теста. |
| Promo | 7/10 | 8/10 | 8/10 | Нет двухнодового cache-теста и большого bulk import-теста. |
| Reference | 8/10 | 8/10 | 8/10 | Владелец item-данных корректен, но multi-node cache contract не покрыт тестом. |
| Control | 7/10 | 8/10 | 8/10 | Audit пока вызывается отдельно от административной мутации. |
| Tasks | 6/10 | 8/10 | 7/10 | Высокая сложность partner/runtime и много плотных составных литералов. |
| Payment | 8/10 | 8/10 | 8/10 | Глобальные provider/asset/rate mutations отделены в Operational; остается очистить публичные Admin signatures от generated SQLC типов. |

## Исправлено

- Import выполняет preflight до транзакции, пишет ограниченными bulk-батчами и
  сериализуется с admin catalog writes по workspace.
- Export основных каталогов читается из согласованного snapshot.
- Кэшированные каталоги используют version scope; ошибка version bump является
  диагностикой и не превращает успешный DB write в ложный отказ API.
- Callback outbox хранит `workspace_id`; admin read/retry/mark/reset не может
  обратиться к событию другой workspace.
- `RefreshDailyStats` пересчитывает только указанную workspace.
- Глобальные Lua partner scripts убраны из Tasks Admin и доступны через Internal.
- Payment больше не хранит локальный `payment_item`: product rewards содержат
  только opaque item key, а метаданными владеет Reference.
- Глобальные Payment provider/asset/rate mutations перенесены из workspace
  Admin в Operational и исключены из Control workspace access catalog.
- Статичный access manifest Control регистрируется одной транзакцией под одним
  registry lock; публичный `Admin.RegisterMethod` удалён.
- Все фиксированные PostgreSQL-запросы сервисов перенесены в SQLC. Динамический
  SQL остался только там, где динамична доверенная таблица callback store или
  компилируется безопасный bulk import.

## Тесты

Service-level integration и benchmark suite каждого сервиса собран в два
корневых файла: `{service}_test.go` и `{service}_bench_test.go`. Unit-тесты
узких подпакетов остаются рядом с owning package, когда проверяют непубличные
helpers и не относятся к API сервиса.
