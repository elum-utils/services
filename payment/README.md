# Payment Service

Модуль `payment` - единый сервис платежей, каталога товаров и выдачи покупок для разных платежных провайдеров. Его главная задача: хранить один каталог, один жизненный цикл заказа и одну логику fulfillment, независимо от того, чем пользователь платит: RUB, VK Votes, Telegram Stars, TON или TON Jettons.

## Основные возможности

- Единый каталог товаров для всех платежных провайдеров.
- Мультивалютные цены в minor units: RUB, VOTE, XTR, TON и TON Jettons.
- Поддержка fixed-товаров и товаров с покупаемым количеством.
- Безопасные лимиты покупок по пользователю и глобально.
- Отдельная модель заказа, платежной попытки, события провайдера и выдачи товара.
- Идемпотентная обработка webhook/update/on-chain событий.
- Снимок цены и состава товара на момент создания заказа.
- Поддержка purchase keys для оплаты товара за другого пользователя.
- Поддержка подписок и проверки активного entitlement.
- Хранение provider-specific identifiers: payment id, invoice id, charge id, subscription id.
- Сырые платежные события сохраняются отдельно от бизнес-статуса.
- Prepared SQLC repository и общий fallback repository.
- Интеграционные тесты, security-тесты и benchmark suite для SQLC и service-level операций.

## Состав API

Сервис собирается через `payment.New(db)` и предоставляет несколько независимых областей:

- `Asset` - управление активами, валютами и связями provider-asset.
- `Product` - каталог товаров, групп, item-ов, локализаций, цен и purchase keys.
- `Checkout` - создание orders, payment attempts, events и завершение оплаты.
- `Subscription` - проверка активной подписки.
- `YooKassa` - RUB-платежи через YooKassa.
- `Platega` - RUB-платежи через Platega.
- `TelegramStars` - платежи Telegram Stars / XTR.
- `TON` - native TON и TON Jetton-платежи.
- `VKMA` - платежи VK Mini Apps / Direct Games через VOTE.

## Каталог

Каталог отделен от платежных провайдеров. Один и тот же товар может продаваться через разные каналы и в разных активах.

Возможности каталога:

- группы товаров;
- товары с названием, описанием, изображением, ссылкой, размером/лейблом;
- период доступности товара;
- позиционирование для витрин;
- флаг видимости;
- флаг закрытого товара;
- локализация названий и описаний;
- item-ы выдачи;
- связь товара с одним или несколькими item-ами;
- количество item-ов в составе товара;
- цены по разным активам;
- промо-цены и временные окна цены;
- кеш каталога для быстрых чтений витрины и checkout.

## Типы товаров

Товар может быть одного из двух режимов количества:

- `fixed` - обычный товар, покупается только одной единицей;
- `flexible` - количественный товар, при оплате можно указать нужное количество.

Безопасный дефолт - `fixed`. Это защищает лимитированные товары от покупки пачкой через `Quantity > 1`.

Для `flexible` товара цена в каталоге считается ценой за одну единицу. При создании заказа итоговая сумма, скидка и payable amount умножаются на выбранное количество. Snapshot item-ов также умножается на выбранное количество.

## Активы и валюты

Сервис хранит активы отдельно от провайдеров. Это позволяет иметь единый каталог цен и явно контролировать, какой провайдер может принимать какой актив.

Поддерживаемые типы активов:

- fiat;
- platform currency;
- crypto native;
- crypto jetton.

Ключевые свойства актива:

- code;
- title;
- kind;
- scale / decimals;
- chain;
- network;
- contract address;
- active flag.

Provider-asset связь задает:

- доступность актива у конкретного провайдера;
- минимальную и максимальную сумму;
- merchant account;
- active flag.

## Цены

Все суммы хранятся в minor units:

- RUB - копейки;
- VOTE - целые голоса;
- XTR - целые Stars;
- TON - nanotons;
- TON Jettons - elementary units по decimals конкретного jetton.

Для каждой цены хранится:

- product id;
- asset code;
- list amount;
- discount amount;
- promotion flag;
- starts at;
- ends at.

Checkout всегда фиксирует snapshot цены в `payment_order`, чтобы последующие изменения каталога не меняли уже созданный заказ.

## Purchase Keys

Purchase key позволяет создать оплату товара за другого пользователя без раскрытия его идентификатора покупателю.

Возможности:

- хранение только hash ключа;
- привязка ключа к workspace, app, platform user и product;
- одноразовые и многоразовые ключи;
- expiration;
- счетчик использований;
- отдельный payer в order;
- лимиты считаются по получателю ключа.

## Checkout Lifecycle

Сервис разделяет бизнес-заказ и платежную попытку.

`payment_order` - бизнес-заказ:

- пользователь;
- получатель;
- товар;
- выбранное количество;
- price snapshot;
- asset;
- locale;
- status;
- reserved until;
- paid at;
- fulfilled at.

`payment_attempt` - конкретная платежная попытка:

- provider code;
- provider payment id;
- provider invoice id;
- provider charge id;
- provider subscription id;
- idempotency key;
- confirmation URL;
- return URL;
- expires at;
- amount snapshot;
- status.

Один order может иметь несколько attempts, если пользователь пересоздает платеж, ссылка истекла или провайдер вернул новый transaction id.

## События Провайдеров

`payment_event` хранит входящие provider events отдельно от order/attempt.

Это нужно для:

- audit trail;
- идемпотентности webhook-ов;
- безопасного retry;
- хранения raw payload hash;
- хранения provider event id;
- фиксации результата проверки подписи;
- диагностики спорных платежей.

## Fulfillment

Выдача товара вынесена в отдельный слой:

- `payment_fulfillment`;
- `payment_fulfillment_item`.

Fulfillment создается только после подтвержденной оплаты. Состав выдачи берется не из текущего каталога, а из snapshot `payment_order_item`, созданного на момент order creation.

Это защищает от ситуаций, когда товар или его состав изменился между созданием платежа и успешным webhook-ом.

## Лимиты

Сервис поддерживает лимиты товара:

- глобальные;
- пользовательские;
- по workspace;
- по platform;
- по platform user;
- по product;
- по временным окнам.

Поддерживаемые интервалы:

- SECOND;
- MINUTE;
- HOUR;
- DAY;
- WEEK;
- MONTH;
- ONCE;
- UNLIMITED.

Для `fixed` товаров лимит расходуется как одна покупка.

Для `flexible` товаров лимит расходуется по выбранному quantity. Например, если лимит 500 coin в день, заказ на quantity 100 расходует 100 единиц лимита.

Инкремент лимита выполняется атомарно при успешной оплате. Счетчик не должен превышать лимит даже при конкурентной обработке платежей.

## Идемпотентность

Сервис рассчитан на повторные события от провайдеров.

Идемпотентность обеспечивается через:

- unique idempotency key;
- unique provider payment id;
- unique provider charge id;
- сохраненные provider events;
- проверку уже fulfilled attempt;
- блокировку order и attempt в транзакции;
- единый `CompleteAttempt`;
- повторный возврат результата без повторной выдачи.

## Подписки

Сервис хранит provider subscriptions и поддерживает проверку активного состояния.

Поддерживаемые свойства:

- provider code;
- workspace;
- provider subscription id;
- app id;
- platform id;
- platform user id;
- product id;
- linked order;
- linked attempt;
- status;
- started at;
- ended at;
- cancel reason.

VKMA и Telegram Stars имеют отдельные adapter-level операции для subscription lifecycle.

## Провайдеры

### YooKassa

Возможности:

- создание RUB-платежа;
- redirect confirmation;
- idempotency key;
- metadata order binding;
- webhook handling;
- sync payment status;
- amount/currency verification;
- provider payment id storage.

Оплата считается успешной только после server-side подтверждения, а не после возврата пользователя на return URL.

### Platega

Возможности:

- создание transaction;
- payment URL / redirect URL;
- H2H transaction details;
- sync transaction status;
- callback handling;
- merchant credential validation;
- amount/currency verification;
- transaction id storage.

### Telegram Stars

Возможности:

- создание invoice link;
- pre-checkout validation;
- successful payment handling;
- Telegram payment charge id storage;
- refund through Bot API;
- subscription edit support;
- XTR amount verification.

Pre-checkout не считается оплатой. Выдача происходит только после successful payment.

### TON

Возможности:

- создание TON payment request;
- native TON payment URI;
- уникальный comment/order payload;
- обработка входящего transfer;
- idempotent transaction storage;
- wallet cursor storage;
- subscriber для входящих транзакций;
- network-aware processing;
- amount/comment/asset verification.

### TON Jettons

Возможности:

- обработка TEP-74 style jetton transfers;
- разбор jetton body;
- хранение sender;
- decimals через asset scale;
- asset code mapping;
- transaction hash deduplication;
- support for USDT_TON, MEMCOIN_TON и других jettons.

### VKMA

Возможности:

- get item response для VK order box;
- chargeable flow;
- subscription status flow;
- VOTE asset;
- workspace-aware callback handling;
- provider order id idempotency;
- automatic fulfillment on successful chargeable event.

## Безопасность

Ключевые инварианты:

- каталог не зависит от провайдера;
- provider callback не выдает товар напрямую;
- выдача происходит только через `CompleteAttempt`;
- amount и asset сверяются со snapshot заказа;
- provider id сверяется со stored attempt;
- fixed-товары нельзя купить количеством больше 1;
- flexible-товары расходуют лимиты по quantity;
- повторный webhook не создает повторную выдачу;
- checkout использует snapshot цены и item-ов;
- raw events сохраняются отдельно;
- purchase keys хранятся только в hash-виде;
- order и attempt блокируются в транзакции при completion.

## Производительность

Сервис использует:

- SQLC generated queries;
- prepared query repository;
- product cache;
- отдельные индексы для order, attempt, product cache, paid order index и limit counters;
- benchmark suite для service-level методов;
- benchmark suite для SQLC queries;
- constraint benchmarks для оценки стоимости foreign keys и индексов.

## Тестовое покрытие

В `payment/tests` есть сценарии для:

- bootstrap схемы и seed данных;
- CRUD assets;
- catalog/checkout/fulfillment cycle;
- лимитов и security cases;
- workspace isolation;
- product cache consistency;
- provider adapters;
- TON full cycle and cursor;
- TON Jetton transfer;
- Telegram Stars payments and subscriptions;
- VKMA flow;
- YooKassa;
- Platega;
- latency и throughput benchmarks.

## Границы Модуля

Payment service отвечает за:

- хранение каталога;
- расчет цены;
- создание заказа;
- создание платежной попытки;
- проверку входящих платежных событий;
- fulfillment snapshot;
- лимиты покупок;
- provider-specific payment lifecycle.

Payment service не должен:

- доверять client-side факту оплаты;
- выдавать товар до server-side подтверждения;
- пересчитывать старый заказ по новой цене каталога;
- позволять quantity для fixed-товаров;
- смешивать provider callback handling и бизнес-выдачу без order/attempt verification.
