# TODO: Promo

## P2

- [x] Добавить test двух нод с общим L2 и независимыми L1 для catalog cache.
- [x] Добавить конкурентный import/admin-write test для promo, localization и
  reward под единым workspace advisory lock.
- [x] Добавить import test, превышающий 60 000 SQL-параметров.

## P3

- [x] Собрать тесты в `promo_test.go` и `promo_bench_test.go`, удалить
  `promo/tests`.
- [ ] Разнести оставшиеся длинные сигнатуры и составные литералы.
