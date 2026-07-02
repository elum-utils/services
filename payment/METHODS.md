# Payment methods

Только методы слоев `user` и `admin`, которые можно использовать как основу будущего API. Адаптеры провайдеров, lifecycle и фоновые операции здесь не описаны.

## user

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `User.ListProducts(ctx, params)` | `ListProductsParams{Identity, GroupCode, AssetCode, Locale}`; `Identity{WorkspaceID, AppID, PlatformID, Platform, PlatformUserID, IsPremium, Sex, Country}`. | Возвращает доступные пользователю продукты с учетом опционального фильтра `GroupCode`, target-фильтра, цены, лимитов и item-ов. |
| `User.GetProduct(ctx, params)` | `GetProductParams{Identity, ProductID, AssetCode, Locale}`. | Возвращает один продукт для пользователя с учетом target-фильтра. |
| `User.GetProductByKey(ctx, params)` | `GetProductByKeyParams{Key, AssetCode, Locale}`. | Возвращает продукт по purchase key. |
| `User.ListAssets(ctx, params)` | `ListAssetsParams{}`. | Возвращает список платежных assets. |
| `User.GetUSDTPrice(ctx, params)` | `GetUSDTPriceParams{AssetCode}`. | Возвращает курс asset к USDT. |
| `User.ListUSDTPrices(ctx, params)` | `ListUSDTPricesParams{}`. | Возвращает список курсов assets к USDT. |
| `User.CreateOrder(ctx, params)` | `CreateOrderParams{Identity, InternalUserID, Payer, ProductID, Quantity, AssetCode, Locale, ReservedUntil, ExpiresAt}`; `Payer{PlatformID, Platform, PlatformUserID, InternalUserID}`. | Создает платежный заказ по продукту. |
| `User.CreateOrderByKey(ctx, params)` | `CreateOrderByKeyParams{Key, Payer, AssetCode, Quantity, Locale, ReservedUntil, ExpiresAt}`. | Создает заказ по purchase key. |
| `User.CreateAttempt(ctx, params)` | `CreateAttemptParams{OrderID, ProviderCode, AssetCode, AmountMinor, ProviderPaymentID, ProviderInvoiceID, ProviderChargeID, ProviderSubscriptionID, IdempotencyKey, ConfirmationURL, ReturnURL, ExpiresAt}`. | Создает попытку оплаты для заказа. |
| `User.IsSubscriptionActive(ctx, params)` | `IsSubscriptionActiveParams{Identity, ProductID, ProviderCode}`. | Проверяет, активна ли подписка пользователя на продукт у провайдера. |

## admin

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `Admin.SaveProduct(ctx, params)` | `SaveProductParams` alias `product.UpsertParams`. | Создает или обновляет продукт через удобный admin-command. |
| `Admin.SaveProductGroup(ctx, params)` | `SaveProductGroupParams` alias `product.UpsertGroupParams`. | Создает или обновляет группу продуктов. |
| `Admin.SaveLocalization(ctx, params)` | `SaveLocalizationParams` alias `product.UpsertLocalizationParams`. | Создает или обновляет локализацию каталога. |
| `Admin.SaveItem(ctx, params)` | `SaveItemParams` alias `product.UpsertItemParams`. | Создает или обновляет item каталога. |
| `Admin.AttachProductItem(ctx, params)` | `AttachProductItemParams` alias `product.AddItemParams`. | Привязывает item к продукту. |
| `Admin.CreateCatalogPrice(ctx, params)` | `CreateCatalogPriceParams` alias `product.CreatePriceParams`. | Создает цену продукта. |
| `Admin.UpdateCatalogPrice(ctx, params)` | `UpdateCatalogPriceParams` alias `product.UpdatePriceParams`. | Обновляет цену продукта. |
| `Admin.Export(ctx, workspaceID, req)` | `workspaceID`, `ExportRequest{Now}`. | Экспортирует каталог payment workspace в `payment.export.v1`: группы, продукты, target, локализации, связи product-item и цены. Reward/reference items не выводятся в payment export. |
| `Admin.PreviewImport(ctx, workspaceID, pkg)` | `workspaceID`, `ExportPackage`. | Проверяет пакет импорта, считает элементы и возвращает конфликты по group/product/item ключам без записи данных. |
| `Admin.Import(ctx, workspaceID, req)` | `ImportRequest{Package, ConflictStrategy}`; стратегии `fail_on_conflict`, `skip_existing`, `update_existing`. | Импортирует каталог payment пачками в транзакции, затем один раз перестраивает product cache workspace. |
| `Admin.ConfigureAsset(ctx, params)` | `ConfigureAssetParams` alias `asset.UpsertParams`. | Создает или обновляет платежный asset. |
| `Admin.ConfigureProviderAsset(ctx, params)` | `ConfigureProviderAssetParams` alias `asset.ProviderUpsertParams`. | Создает или обновляет связь провайдера и asset. |
| `Admin.ListProductGroups(ctx, params)` | `ProductGroupListParams{WorkspaceID, Page}`. | Возвращает группы продуктов. |
| `Admin.GetProductGroup(ctx, workspaceID, code)` | `workspaceID`, `code`. | Возвращает группу продуктов. |
| `Admin.UpsertProductGroup(ctx, params)` | `paymentsqlc.UpsertProductGroupParams`. | Создает или обновляет группу продуктов. |
| `Admin.DeleteProductGroup(ctx, workspaceID, code)` | `workspaceID`, `code`. | Удаляет группу продуктов. |
| `Admin.ListLocalizations(ctx, params)` | `LocalizationListParams{WorkspaceID, Locale, Page}`. | Возвращает локализации каталога. |
| `Admin.GetLocalization(ctx, workspaceID, locale, key)` | `workspaceID`, `locale`, `key`. | Возвращает локализацию каталога. |
| `Admin.UpsertLocalization(ctx, params)` | `paymentsqlc.UpsertLocalizationParams`. | Создает или обновляет локализацию. |
| `Admin.DeleteLocalization(ctx, workspaceID, locale, key)` | `workspaceID`, `locale`, `key`. | Удаляет локализацию. |
| `Admin.ListProducts(ctx, params)` | `ProductListParams{WorkspaceID, GroupCode, QuantityMode, Page}`. | Возвращает продукты каталога. |
| `Admin.GetProduct(ctx, workspaceID, id)` | `workspaceID`, `id`. | Возвращает продукт. |
| `Admin.UpsertProduct(ctx, params)` | `paymentsqlc.UpsertProductParams`. | Создает или обновляет продукт. |
| `Admin.DeleteProduct(ctx, workspaceID, id)` | `workspaceID`, `id`. | Удаляет продукт. |
| `Admin.ListItems(ctx, params)` | `ItemListParams{WorkspaceID, ItemType, Page}`. | Возвращает items каталога. |
| `Admin.GetItem(ctx, workspaceID, id)` | `workspaceID`, `id`. | Возвращает item. |
| `Admin.UpsertItem(ctx, params)` | `paymentsqlc.UpsertItemParams`. | Создает или обновляет item. |
| `Admin.DeleteItem(ctx, workspaceID, id)` | `workspaceID`, `id`. | Удаляет item. |
| `Admin.ListProductItems(ctx, params)` | `ProductItemListParams{WorkspaceID, ProductID, ItemID, Page}`. | Возвращает связи продуктов и items. |
| `Admin.UpsertProductItem(ctx, params)` | `paymentsqlc.UpsertProductItemParams`. | Создает или обновляет связь продукта и item. |
| `Admin.DeleteProductItem(ctx, workspaceID, productID, itemID)` | `workspaceID`, `productID`, `itemID`. | Удаляет связь продукта и item. |
| `Admin.ListPrices(ctx, params)` | `PriceListParams{WorkspaceID, ProductID, AssetCode, Page}`. | Возвращает цены продуктов. |
| `Admin.GetPrice(ctx, workspaceID, id)` | `workspaceID`, `id`. | Возвращает цену. |
| `Admin.CreatePrice(ctx, params)` | `paymentsqlc.CreateProductPriceParams`. | Создает цену. |
| `Admin.UpdatePrice(ctx, params)` | `paymentsqlc.UpdateProductPriceParams`. | Обновляет цену. |
| `Admin.DeletePrice(ctx, workspaceID, id)` | `workspaceID`, `id`. | Удаляет цену. |
| `Admin.ListProductLimitCounters(ctx, params)` | `ProductLimitCounterListParams{WorkspaceID, ProductID, PlatformID, PlatformUserID, Page}`. | Возвращает счетчики лимитов продуктов. |
| `Admin.DeleteProductLimitCounter(ctx, params)` | `paymentsqlc.AdminDeleteProductLimitCounterParams`. | Удаляет счетчик лимита продукта. |
| `Admin.ListProviders(ctx)` | Только `ctx`. | Возвращает платежных провайдеров. |
| `Admin.GetProvider(ctx, code)` | `code`. | Возвращает провайдера. |
| `Admin.UpsertProvider(ctx, params)` | `ProviderUpsertParams{Code, Title, ProviderKind, SupportsCreate, SupportsRedirect, SupportsWebhook, SupportsRefund, IsActive}`. | Создает или обновляет провайдера. |
| `Admin.DeleteProvider(ctx, code)` | `code`. | Удаляет провайдера. |
| `Admin.ListAssets(ctx)` | Только `ctx`. | Возвращает assets. |
| `Admin.GetAsset(ctx, code)` | `code`. | Возвращает asset. |
| `Admin.UpsertAsset(ctx, params)` | `paymentsqlc.UpsertAssetParams`. | Создает или обновляет asset. |
| `Admin.DeleteAsset(ctx, code)` | `code`. | Удаляет asset. |
| `Admin.ListProviderAssets(ctx, params)` | `ProviderAssetListParams{ProviderCode, AssetCode, Page}`. | Возвращает связи провайдеров и assets. |
| `Admin.GetProviderAsset(ctx, providerCode, assetCode)` | `providerCode`, `assetCode`. | Возвращает связь провайдера и asset. |
| `Admin.UpsertProviderAsset(ctx, params)` | `paymentsqlc.UpsertProviderAssetParams`. | Создает или обновляет связь провайдера и asset. |
| `Admin.DeleteProviderAsset(ctx, providerCode, assetCode)` | `providerCode`, `assetCode`. | Удаляет связь провайдера и asset. |
| `Admin.SaveTONWallet(ctx, params)` | `TONWalletUpsertParams{WorkspaceID, Network, WalletAddress, NetworkConfigURL, IsEnabled}`. | Создает или обновляет TON merchant wallet workspace для автоматической подписки на входящие платежи. `NetworkConfigURL` опционален, по умолчанию берется config mainnet/testnet. |
| `Admin.ListTONWallets(ctx, params)` | `TONWalletListParams{WorkspaceID, Network, IsEnabled, Page}`. | Возвращает TON wallets workspace с опциональным фильтром по network и enabled. |
| `Admin.GetTONWallet(ctx, workspaceID, network, walletAddress)` | `workspaceID`, `network`, `walletAddress`. | Возвращает конфигурацию TON wallet. |
| `Admin.DeleteTONWallet(ctx, workspaceID, network, walletAddress)` | `workspaceID`, `network`, `walletAddress`. | Удаляет TON wallet; runtime-синхронизация автоматически остановит соответствующий subscriber. |
| `Admin.GetAssetRate(ctx, assetCode, referenceAssetCode)` | `assetCode`, `referenceAssetCode`. | Возвращает курс asset к reference asset. |
| `Admin.ListAssetRates(ctx, params)` | `AssetRateListParams{AssetCode, ReferenceAssetCode, Page}`. | Возвращает список курсов assets. |
| `Admin.UpdateAssetRate(ctx, params)` | `UpdateAssetRateParams{AssetCode, ReferenceAssetCode, ReferencePerAssetMinor, Source, ObservedAt}`. | Обновляет курс asset и связанные динамические цены. |
| `Admin.ConfigureAssetRateAutoUpdate(ctx, params)` | `ConfigureAssetRateAutoUpdateParams{AssetCode, ReferenceAssetCode, Enabled, Source, SourceChainID, SourceTokenAddress}`. | Настраивает автоматическое обновление курса. |
| `Admin.CreateProductKey(ctx, params)` | `CreateProductKeyParams`. | Создает purchase key для продукта. |
| `Admin.RebuildProductCache(ctx, workspaceID)` | `workspaceID`. | Пересобирает кеш каталога продуктов рабочей области. |
| `Admin.ListPurchaseKeys(ctx, params)` | `PurchaseKeyListParams{WorkspaceID, ProductID, Status, PlatformID, PlatformUserID, Page}`. | Возвращает purchase keys. |
| `Admin.GetPurchaseKey(ctx, workspaceID, id)` | `workspaceID`, `id`. | Возвращает purchase key. |
| `Admin.UpdatePurchaseKeyStatus(ctx, workspaceID, id, status)` | `workspaceID`, `id`, `status`. | Обновляет статус purchase key. |
| `Admin.ListOrders(ctx, params)` | `OrderListParams{WorkspaceID, Status, ProductID, PlatformID, PlatformUserID, Page}`. | Возвращает платежные заказы. |
| `Admin.GetOrder(ctx, id)` | `id`. | Возвращает заказ. |
| `Admin.GetOrderByPublicID(ctx, publicID)` | `publicID`. | Возвращает заказ по публичному id. |
| `Admin.UpdateOrderStatus(ctx, workspaceID, id, status)` | `workspaceID`, `id`, `status`. | Обновляет статус заказа. |
| `Admin.ListPaymentAttempts(ctx, params)` | `AttemptListParams{WorkspaceID, OrderID, ProviderCode, Status, Page}`. | Возвращает попытки оплаты. |
| `Admin.GetPaymentAttempt(ctx, id)` | `id`. | Возвращает попытку оплаты. |
| `Admin.UpdatePaymentAttemptStatus(ctx, id, status)` | `id`, `status`. | Обновляет статус попытки оплаты. |
| `Admin.CreatePaymentEvent(ctx, params)` | `CreatePaymentEventParams`. | Создает платежное событие. |
| `Admin.ListPaymentEvents(ctx, params)` | `EventListParams{WorkspaceID, ProviderCode, ProcessingStatus, Page}`. | Возвращает платежные события. |
| `Admin.GetPaymentEvent(ctx, id)` | `id`. | Возвращает платежное событие. |
| `Admin.UpdatePaymentEventProcessingStatus(ctx, id, status, message)` | `id`, `status`, `message`. | Обновляет статус обработки платежного события. |
| `Admin.CompletePaymentAttempt(ctx, params)` | `CompletePaymentAttemptParams`. | Завершает попытку оплаты и запускает fulfillment. |
| `Admin.ListSubscriptions(ctx, params)` | `SubscriptionListParams{WorkspaceID, ProviderCode, ProductID, Status, PlatformID, PlatformUserID, Page}`. | Возвращает подписки. |
| `Admin.GetSubscription(ctx, workspaceID, id)` | `workspaceID`, `id`. | Возвращает подписку. |
| `Admin.GetSubscriptionByProviderID(ctx, providerCode, providerSubscriptionID)` | `providerCode`, `providerSubscriptionID`. | Возвращает подписку по id провайдера. |
| `Admin.UpsertSubscription(ctx, params)` | `paymentsqlc.UpsertPaymentSubscriptionParams`. | Создает или обновляет подписку. |
| `Admin.UpdateSubscriptionStatus(ctx, params)` | `paymentsqlc.UpdatePaymentSubscriptionStatusParams`. | Обновляет статус подписки. |
| `Admin.ListFulfillments(ctx, params)` | `FulfillmentListParams{WorkspaceID, Status, OrderID, Page}`. | Возвращает fulfillments. |
| `Admin.GetFulfillment(ctx, id)` | `id`. | Возвращает fulfillment. |
| `Admin.UpdateFulfillmentStatus(ctx, id, status, message)` | `id`, `status`, `message`. | Обновляет статус fulfillment. |
| `Admin.ListFulfillmentItems(ctx, params)` | `FulfillmentItemListParams{WorkspaceID, FulfillmentID, Page}`. | Возвращает fulfillment items. |
| `Admin.CreateRefund(ctx, params)` | `RefundCreateParams{OrderID, AttemptID, ProviderCode, ProviderRefundID, AmountMinor, AssetCode, Status, Reason}`. | Создает запись refund. |
| `Admin.ExecuteRefund(ctx, params)` | `ExecuteRefundParams`. | Выполняет refund через платежный слой и возвращает результат. |
| `Admin.ListRefunds(ctx, params)` | `RefundListParams{WorkspaceID, OrderID, ProviderCode, Status, Page}`. | Возвращает refunds. |
| `Admin.GetRefund(ctx, id)` | `id`. | Возвращает refund. |
| `Admin.UpdateRefundStatus(ctx, id, status, reason)` | `id`, `status`, `reason`. | Обновляет статус refund. |
| `Admin.GetStats(ctx, workspaceID)` | `workspaceID`. | Возвращает общую статистику платежей. |
| `Admin.GetProductStats(ctx, workspaceID, productID)` | `workspaceID`, `productID`. | Возвращает статистику продукта. |
| `Admin.ListDailyStats(ctx, workspaceID, productID, from, until)` | `workspaceID`, `productID`, `from`, `until`. | Возвращает дневную статистику. |
| `Admin.ListDailyOverview(ctx, workspaceID, from, until)` | `workspaceID`, `from`, `until`. | Возвращает дневный обзор платежей. |
| `Admin.RefreshDailyStats(ctx, from, until)` | `from`, `until`. | Пересчитывает дневную статистику. |
| `Admin.ListProviderCursors(ctx, params)` | `ProviderCursorListParams{WorkspaceID, ProviderCode, Network, Page}`. | Возвращает курсоры синхронизации провайдеров. |
| `Admin.GetProviderCursor(ctx, workspaceID, providerCode, network, sourceKey)` | `workspaceID`, `providerCode`, `network`, `sourceKey`. | Возвращает курсор провайдера. |
| `Admin.UpsertProviderCursor(ctx, params)` | `paymentsqlc.UpsertProviderCursorParams`. | Создает или обновляет курсор провайдера. |
| `Admin.ListProviderTransactions(ctx, params)` | `ProviderTransactionListParams{WorkspaceID, ProviderCode, Network, SourceKey, Status, Page}`. | Возвращает транзакции провайдеров. |
| `Admin.GetProviderTransaction(ctx, workspaceID, id)` | `workspaceID`, `id`. | Возвращает транзакцию провайдера. |
| `Admin.GetProviderTransactionByExternalID(ctx, workspaceID, providerCode, network, sourceKey, externalTransactionID)` | `workspaceID`, `providerCode`, `network`, `sourceKey`, `externalTransactionID`. | Возвращает транзакцию по внешнему id. |
| `Admin.UpdateProviderTransactionStatus(ctx, workspaceID, id, status, message)` | `workspaceID`, `id`, `status`, `message`. | Обновляет статус транзакции провайдера. |
| `Admin.ListCallbackEvents(ctx, params)` | `CallbackEventListParams{SourceService, EventType, Status, Page}`. | Возвращает callback-события payment. |
| `Admin.GetCallbackEvent(ctx, id)` | `id`. | Возвращает callback-событие. |
| `Admin.RetryCallbackEventNow(ctx, id)` | `id`. | Отправляет callback-событие на повторную обработку. |
| `Admin.MarkCallbackEventOK(ctx, id)` | `id`. | Помечает callback-событие успешным. |
| `Admin.MarkCallbackEventReject(ctx, id, reason)` | `id`, `reason`. | Помечает callback-событие отклоненным. |
| `Admin.ResetExpiredCallbackProcessing(ctx)` | Только `ctx`. | Сбрасывает зависшие callback-события в обработке. |
