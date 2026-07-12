# TODO: Control

## P1

- [x] Сделать security audit обязательной частью административной мутации.
  Service добавляет audit metadata автоматически, а repository записывает
  изменение и `control_audit_event` в одной PostgreSQL-транзакции.

## P2

- [x] Добавить конкурентный test регистрации конфликтующих manifests; текущий
  test покрывает atomic rollback одного manifest.
- [x] Определить отдельный global/platform scope RBAC до выдачи прав на
  глобальные каталоги других сервисов.

## P3

- [x] Собрать тесты в `control_test.go` и `control_bench_test.go`, удалить
  `control/tests`.
- [ ] Разнести длинные repository/service literals и вызовы по одному полю или
  аргументу на строку.
