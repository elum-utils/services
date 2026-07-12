# TODO: Payment

## P1

- [x] Удалить локальный item catalog из публичного Payment API и product cache.
  `payment_item`, `Admin.UpsertItem/ListItems/GetItem/DeleteItem` и поля
  `ProductItem.Type/Title/Description/Rarity/Position` дублируют Reference и
  нарушают зафиксированный ownership contract.
- [x] Определить global/platform RBAC для provider, asset и asset-rate catalog
  либо сделать эти данные workspace-scoped. Текущий Control RBAC проверяется в
  workspace, а Payment mutation изменяет глобальные строки.

## P2

- [x] Добавить test двух нод для product/limit config version cache.
- [x] Добавить конкурентный import/admin-write test и большой import test,
  превышающий 60 000 SQL-параметров.

## P3

- [x] Собрать тесты в `payment_test.go` и `payment_bench_test.go`, удалить
  `payment/tests`.
- [ ] Убрать generated SQLC types из публичных Admin signatures и разнести
  оставшиеся длинные literals/calls по одному полю или аргументу на строку.
