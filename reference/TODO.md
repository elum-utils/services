# TODO: Reference

## P2

- [x] Добавить test двух нод с общим L2 и независимыми L1 для `Get`, `Resolve`
  и admin reads после item/localization mutation.
- [x] Добавить конкурентный import/admin-write test и большой import test,
  превышающий 60 000 SQL-параметров.

## P3

- [x] Собрать тесты в `reference_test.go` и `reference_bench_test.go`, удалить
  `reference/tests`.
- [x] Разнести оставшиеся длинные сигнатуры и составные литералы.
