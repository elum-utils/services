# Control methods

Только методы слоя `admin`, которые можно использовать как основу будущего API.

## admin

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `Admin.CompleteAuth(ctx, params)` | `AuthIdentityParams{Provider, Subject, DisplayName, Payload, IP, UserAgent, BindToIP, ExpiresAt}`. | Принимает identity от уже проверенного OAuth-adapter-а, находит или создаёт account и создаёт session либо 2FA challenge. |
| `Admin.CompleteTwoFactor(ctx, challenge, code, ip)` | Одноразовый challenge token, TOTP/backup-код, IP. | Завершает вход с 2FA и выдаёт session token. |
| `Admin.GetAccount(ctx, accountID)` | `accountID` оператора. | Возвращает профиль оператора. |
| `Admin.ListIdentities(ctx, accountID)` | `accountID` текущего оператора. | Возвращает внешние аккаунты, привязанные к оператору. |
| `Admin.BindIdentity(ctx, accountID, params)` | `accountID`, проверенная `AuthIdentityParams`. | Привязывает дополнительный аккаунт GitHub, GitLab, Google, VK или Yandex. |
| `Admin.UnbindIdentity(ctx, accountID, provider)` | `accountID`, provider. | Отвязывает provider, если у account останется хотя бы один способ входа. |
| `Admin.ValidateSession(ctx, token, ip)` | Session token, IP. | Проверяет сессию, срок действия и IP binding. |
| `Admin.BeginTwoFactor(ctx, accountID, issuer)` | `accountID`, issuer. | Создаёт TOTP secret и URI для подключения 2FA. |
| `Admin.ConfirmTwoFactor(ctx, accountID, code)` | `accountID`, первый TOTP-код. | Активирует 2FA и возвращает одноразовые backup-коды. |
| `Admin.DisableTwoFactor(ctx, accountID, code)` | `accountID`, TOTP или backup-код. | Отключает 2FA после подтверждения. |
| `Admin.ListSessions(ctx, accountID)` | `accountID`. | Возвращает активные сессии оператора без токенов. |
| `Admin.RevokeSession(ctx, accountID, sessionID)` | `accountID`, `sessionID`. | Отзывает одну сессию оператора. |
| `Admin.RevokeAllSessions(ctx, accountID, exceptSessionID)` | `accountID`, опциональный `exceptSessionID`. | Отзывает все сессии, кроме при необходимости текущей. |
| `Admin.ListWorkspaces(ctx, accountID, limit, offset)` | `accountID`, пагинация. | Возвращает workspace, в которых оператор является активным участником. |
| `Admin.AcceptInvite(ctx, accountID, token)` | `accountID`, исходный invite token. | Принимает invite и добавляет оператора в workspace с заданными ролями. |
| `Admin.CreateWorkspace(ctx, params)` | `CreateWorkspaceParams{ActorID, ID, Slug, Title}`. | Создаёт workspace, добавляет создателя и системную owner-роль. |
| `Admin.GetWorkspace(ctx, workspaceID)` | `workspaceID`. | Возвращает workspace. |
| `Admin.UpdateWorkspace(ctx, params)` | `UpdateWorkspaceParams{ActorID, WorkspaceID, Slug, Title, Status}`. | Обновляет реквизиты или архивирует workspace после проверки права. |
| `Admin.ListMembers(ctx, workspaceID, limit, offset)` | `workspaceID`, пагинация. | Возвращает активных участников и назначенные роли. |
| `Admin.RemoveMember(ctx, actorID, workspaceID, accountID)` | Actor, workspace и участник. | Удаляет участника при наличии метода и строгом превосходстве роли actor. |
| `Admin.CreateInvite(ctx, params)` | `CreateInviteParams{ActorID, WorkspaceID, RoleIDs, ExpiresAt, MaxUses}`. | Создаёт invite с ролями, сроком действия и лимитом использований. |
| `Admin.ListInvites(ctx, workspaceID, limit, offset)` | `workspaceID`, пагинация. | Возвращает приглашения workspace без исходного токена. |
| `Admin.RevokeInvite(ctx, actorID, workspaceID, inviteID)` | Actor, workspace и invite. | Отзывает invite после проверки права и иерархии его ролей. |
| `Admin.CreateRole(ctx, params)` | `CreateRoleParams{ActorID, WorkspaceID, Code, Title, Description, Position}`. | Создаёт роль строго ниже высшей роли actor. |
| `Admin.UpdateRole(ctx, params)` | `UpdateRoleParams{ActorID, WorkspaceID, RoleID, Title, Description, Position}`. | Обновляет роль только если она строго ниже actor. |
| `Admin.DeleteRole(ctx, actorID, workspaceID, roleID)` | Actor, workspace и роль. | Удаляет роль и её назначения, если она строго ниже actor. |
| `Admin.ListRoles(ctx, workspaceID)` | `workspaceID`. | Возвращает роли workspace с количеством участников и включёнными methods. |
| `Admin.SetRoleMember(ctx, params)` | `SetRoleMemberParams{ActorID, WorkspaceID, AccountID, RoleID}`. | Назначает роль, если target account и role строго ниже actor. |
| `Admin.RemoveRoleMember(ctx, params)` | `RemoveRoleMemberParams{ActorID, WorkspaceID, AccountID, RoleID}`. | Снимает роль с участника при той же проверке иерархии. |
| `Admin.ListRolePermissions(ctx, workspaceID, roleID)` | `workspaceID`, `roleID`. | Возвращает включённые method keys роли. |
| `Admin.SetRolePermission(ctx, params)` | `SetRolePermissionParams{ActorID, WorkspaceID, RoleID, MethodKey, Enabled}`. | Включает или выключает method key у роли, если роль строго ниже actor. |
| `Admin.ClearRolePermissions(ctx, params)` | `ClearRolePermissionsParams{ActorID, WorkspaceID, RoleID}`. | Удаляет все включённые methods роли, если роль строго ниже actor. |
| `Admin.ListMethods(ctx)` | Нет параметров. | Возвращает все зарегистрированные методы для административного интерфейса. |
| `Admin.GetMethod(ctx, methodKey)` | `methodKey`. | Возвращает публичные метаданные зарегистрированного метода. |
| `Admin.AppendAudit(ctx, params)` | `AuditEventParams{WorkspaceID, ActorID, MethodKey, TargetType, TargetID, BeforeData, AfterData, Result, RequestID}`. | Добавляет событие аудита после административного действия. |
| `Admin.ListAudit(ctx, workspaceID, page)` | `workspaceID`, `Page{Limit, Offset}`. | Возвращает аудит workspace с пагинацией. |

## adapters

Это не доменный слой и не отдельный API control, а готовые функции для
REST/gateway слоя. Они скрывают provider-specific авторизацию и возвращают
`admin.AuthIdentityParams`, которые REST handler уже сам передаёт в
`Admin.CompleteAuth`.

| Метод | Что принимаем | Что делает |
| --- | --- | --- |
| `auth.Google(ctx, params)` | `OAuth2AuthParams{ClientID, ClientSecret, Code, AccessToken, RedirectURI, IP, UserAgent, BindToIP, ExpiresAt, TokenURL, UserInfoURL, Mapping, HTTPClient, Timeout}`. | Проверяет Google OAuth/OpenID Connect ввод и возвращает `admin.AuthIdentityParams` для `Admin.CompleteAuth`. |
| `auth.GitHub(ctx, params)` | `OAuth2AuthParams{ClientID, ClientSecret, Code, AccessToken, RedirectURI, IP, UserAgent, BindToIP, ExpiresAt, TokenURL, UserInfoURL, Mapping, HTTPClient, Timeout}`. | Проверяет GitHub OAuth ввод и возвращает `admin.AuthIdentityParams`. |
| `auth.GitLab(ctx, params)` | `OAuth2AuthParams{ClientID, ClientSecret, Code, AccessToken, RedirectURI, IP, UserAgent, BindToIP, ExpiresAt, TokenURL, UserInfoURL, Mapping, HTTPClient, Timeout}`. | Проверяет GitLab OAuth ввод и возвращает `admin.AuthIdentityParams`. |
| `auth.Yandex(ctx, params)` | `OAuth2AuthParams{ClientID, ClientSecret, Code, AccessToken, RedirectURI, IP, UserAgent, BindToIP, ExpiresAt, TokenURL, UserInfoURL, Mapping, HTTPClient, Timeout}`. | Проверяет Yandex OAuth ввод и возвращает `admin.AuthIdentityParams`. |
| `auth.VKID(ctx, params)` | `OAuth2AuthParams{ClientID, ClientSecret, Code, AccessToken, RedirectURI, IP, UserAgent, BindToIP, ExpiresAt, TokenURL, UserInfoURL, Mapping, HTTPClient, Timeout}`. | Проверяет VK ID ввод и возвращает `admin.AuthIdentityParams`. |
| `auth.TelegramWebApp(ctx, params)` | `TelegramWebAppAuthParams{BotToken, InitData, IP, UserAgent, BindToIP, ExpiresAt, MaxAge, Now}`. | Проверяет Telegram WebApp `initData`, подпись и срок действия, затем возвращает `admin.AuthIdentityParams`. |
| `auth.TONConnectPayload(secret, ttl)` | Secret backend-а и TTL payload-а. | Генерирует payload/nonce для запроса `ton_proof` у кошелька. |
| `auth.TONConnect(ctx, params)` | `TONConnectAuthParams{Address, Network, PublicKey, WalletStateInit, Proof, ExpectedPayload или PayloadSecret, ExpectedDomain, ExpectedNetwork, Client, IP, UserAgent, BindToIP, ExpiresAt, MaxAge}`. | Проверяет TON Connect `ton_proof` по спецификации: payload, domain, timestamp, stateInit/address, wallet public key и Ed25519 signature. Возвращает `admin.AuthIdentityParams`. |
| `auth.New(admin, providers...)` | `control.Admin`-совместимый сервис и список provider-ов. | Создаёт auth adapter, который можно использовать внутри REST handler-а. |
| `Auth.Register(provider)` | Provider с методами `Provider()` и `Resolve(ctx, request)`. | Регистрирует дополнительный способ авторизации. |
| `Auth.Authenticate(ctx, request)` | `Request{Provider, Code, AccessToken, RedirectURI, State, RawData, IP, UserAgent, BindToIP, ExpiresAt}`. | Выбирает provider, проверяет внешний ввод, нормализует identity и вызывает `Admin.CompleteAuth`. |
| `auth.NewGoogle(config)` | `OAuth2ProviderConfig{ClientID, ClientSecret, TokenURL, UserInfoURL, Mapping, HTTPClient, Timeout}`. | Создаёт provider Google OAuth/OpenID Connect с дефолтными endpoint-ами и mapping-ом. |
| `auth.NewGitHub(config)` | `OAuth2ProviderConfig{ClientID, ClientSecret, TokenURL, UserInfoURL, Mapping, HTTPClient, Timeout}`. | Создаёт provider GitHub OAuth с дефолтными endpoint-ами и mapping-ом. |
| `auth.NewGitLab(config)` | `OAuth2ProviderConfig{ClientID, ClientSecret, TokenURL, UserInfoURL, Mapping, HTTPClient, Timeout}`. | Создаёт provider GitLab OAuth с дефолтными endpoint-ами и mapping-ом. |
| `auth.NewYandex(config)` | `OAuth2ProviderConfig{ClientID, ClientSecret, TokenURL, UserInfoURL, Mapping, HTTPClient, Timeout}`. | Создаёт provider Yandex OAuth с дефолтными endpoint-ами и mapping-ом. |
| `auth.NewVKID(config)` | `OAuth2ProviderConfig{ClientID, ClientSecret, TokenURL, UserInfoURL, Mapping, HTTPClient, Timeout}`. | Создаёт отдельный provider VK ID; при отличиях VK flow можно переопределить endpoint-ы или заменить реализацию без изменения `Auth.Authenticate`. |
| `auth.NewOAuth2(config)` | `OAuth2Config{Provider, ClientID, ClientSecret, TokenURL, UserInfoURL, Mapping, HTTPClient, Timeout}`. | Низкоуровневый building block для нестандартных OAuth2-провайдеров. |
| `auth.NewTelegramWebApp(config)` | `TelegramWebAppConfig{Provider, BotToken, MaxAge, Now}`. | Создаёт provider для Telegram WebApp `initData`: проверяет подпись, срок действия и достаёт пользователя. |
