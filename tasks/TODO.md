# TODO: Tasks

## P2

- [x] Добавить test двух нод с общим L2 и независимыми L1 для task catalog и
  partner config version scopes.
- [x] Добавить конкурентный import/admin-write test для group, task,
  localization, reward, complex condition и partner config.
- [x] Добавить import test, превышающий 60 000 SQL-параметров.

## P3

- [x] Собрать тесты в `tasks_test.go` и `tasks_bench_test.go`, удалить
  `tasks/tests`.
- [ ] Разделить плотные literals/mappers и длинные partner/runtime функции на
  читаемые логические блоки без изменения поведения.
