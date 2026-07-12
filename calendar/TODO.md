# TODO: Calendar

## P2

- [x] Добавить test двух нод с общим L2 и независимыми L1: admin mutation на
  первой ноде должна сделать старую версию user/admin catalog недостижимой на
  второй.
- [x] Добавить конкурентный test `Import(fail_on_conflict)` против
  localization/step/reward write и проверить единый workspace advisory lock.
- [x] Добавить import test, превышающий 60 000 SQL-параметров.

## P3

- [x] Собрать тесты в `calendar_test.go` и `calendar_bench_test.go`, после чего
  удалить `calendar/tests`.
- [ ] Разнести оставшиеся длинные сигнатуры и составные литералы по одному
  аргументу или полю на строку.
