package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	"github.com/elum-utils/services/internal/utils/mysqlutil"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"

	serviceerrors "github.com/elum-utils/services/errors"
	"github.com/elum-utils/services/payment/adapters/platega"
	"github.com/elum-utils/services/payment/adapters/telegramstars"
	"github.com/elum-utils/services/payment/adapters/ton"
	"github.com/elum-utils/services/payment/adapters/vkma"
	"github.com/elum-utils/services/payment/adapters/yookassa"
	"github.com/elum-utils/services/payment/repository"
	"github.com/elum-utils/services/payment/service/admin"
	"github.com/elum-utils/services/payment/service/asset"
	"github.com/elum-utils/services/payment/service/checkout"
	"github.com/elum-utils/services/payment/service/product"
	"github.com/elum-utils/services/payment/service/refund"
	"github.com/elum-utils/services/payment/service/subscription"
	"github.com/elum-utils/services/payment/service/user"
)

type Payment struct {
	Admin *admin.Admin
	User  *user.User

	Adapters *Adapters

	asset             *asset.Asset
	product           *product.Product
	checkout          *checkout.Checkout
	refund            *refund.Refund
	subscription      *subscription.Subscription
	callbacks         *callbackutil.Store
	client            *sqlwrap.Client
	ownsClient        bool
	rootCtx           context.Context
	rootCancel        context.CancelFunc
	background        sync.WaitGroup
	pricing           *repository.PaymentRepository
	pricingHTTPClient *http.Client
	pricingInterval   time.Duration
	pricingBaseURL    string
	pricingDone       <-chan struct{}

	lifecycleMu    sync.Mutex
	params         DatabaseParams
	callbacksToRun []callbackRegistration
	running        bool
}

type Adapters struct {
	TON           *ton.TON
	TelegramStars *telegramstars.TelegramStars
	Platega       *platega.Platega
	VKMA          *vkma.VKMA
	YooKassa      *yookassa.YooKassa
}

func New(params DatabaseParams) *Payment {
	return &Payment{params: params}
}

func NewWithDatabase(ctx context.Context, db *sql.DB, options Options) (*Payment, error) {
	client, err := sqlwrap.New(db, toSQLWrapOptions(options))
	if err != nil {
		return nil, serviceerrors.Wrap(serviceerrors.CodeInternalError, "payment sql client initialization failed", err)
	}
	return newAPI(ctx, client, false, options), nil
}

func (a *Payment) Run(ctx context.Context) error {
	if a == nil {
		return ErrServiceNil
	}
	a.lifecycleMu.Lock()
	if a.running {
		a.lifecycleMu.Unlock()
		return ErrServiceRunning
	}
	a.running = true
	params := a.params
	registrations := append([]callbackRegistration(nil), a.callbacksToRun...)
	a.lifecycleMu.Unlock()

	running, err := open(ctx, params)
	if err != nil {
		a.lifecycleMu.Lock()
		a.running = false
		a.lifecycleMu.Unlock()
		if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			return nil
		}
		return wrapLifecycleError(err)
	}
	a.adopt(running)
	defer a.Close()

	errCh := make(chan error, len(registrations))
	a.background.Add(len(registrations))
	for _, registration := range registrations {
		registration := registration
		go func() {
			defer a.background.Done()
			errCh <- a.runCallback(registration.ctx, registration.handler, registration.options...)
		}()
	}

	select {
	case <-a.rootCtx.Done():
		return nil
	case err := <-errCh:
		if errors.Is(err, context.Canceled) && a.rootCtx.Err() != nil {
			return nil
		}
		return wrapLifecycleError(err)
	}
}

func open(ctx context.Context, params DatabaseParams) (*Payment, error) {
	if params.User == "" {
		return nil, ErrDatabaseUserRequired
	}
	if params.Database == "" {
		return nil, ErrDatabaseNameRequired
	}
	db, err := mysqlutil.Open(ctx, mysqlutil.Config{
		User: params.User, Password: params.Password, Database: params.Database,
		Host: params.Host, Port: params.Port,
	})
	if err != nil {
		return nil, serviceerrors.Wrap(serviceerrors.CodeUnavailable, "payment database connection failed", err)
	}
	client, err := sqlwrap.New(db, toSQLWrapOptions(params.Options))
	if err != nil {
		_ = db.Close()
		return nil, serviceerrors.Wrap(serviceerrors.CodeInternalError, "payment sql client initialization failed", err)
	}
	bootstrap := repository.NewPaymentRepositoryWithOptions(client, repository.Options{
		QueryTimeout: params.Options.QueryTimeout,
		CacheL1Delay: params.Options.CacheL1Delay,
		CacheL2Delay: params.Options.CacheL2Delay,
	})
	if err := bootstrap.Bootstrap(ctx); err != nil {
		_ = bootstrap.Close()
		_ = client.Close()
		return nil, serviceerrors.Wrap(serviceerrors.CodeInternalError, "payment bootstrap failed", err)
	}
	if err := bootstrap.Close(); err != nil {
		_ = client.Close()
		return nil, serviceerrors.Wrap(serviceerrors.CodeInternalError, "payment bootstrap shutdown failed", err)
	}
	return newAPI(ctx, client, true, params.Options), nil
}

func (a *Payment) adopt(running *Payment) {
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	a.Admin = running.Admin
	a.User = running.User
	a.asset = running.asset
	a.product = running.product
	a.checkout = running.checkout
	a.refund = running.refund
	a.subscription = running.subscription
	a.Adapters = running.Adapters
	a.callbacks = running.callbacks
	a.client = running.client
	a.ownsClient = running.ownsClient
	a.rootCtx = running.rootCtx
	a.rootCancel = running.rootCancel
	a.pricing = running.pricing
	a.pricingHTTPClient = running.pricingHTTPClient
	a.pricingInterval = running.pricingInterval
	a.pricingBaseURL = running.pricingBaseURL
	a.pricingDone = running.pricingDone
}

func newAPI(ctx context.Context, db *sqlwrap.Client, ownsClient bool, options Options) *Payment {
	rootCtx, rootCancel := context.WithCancel(normalizeLifecycleContext(ctx))
	repositoryOptions := repository.Options{
		QueryTimeout: options.QueryTimeout,
		CacheL1Delay: options.CacheL1Delay,
		CacheL2Delay: options.CacheL2Delay,
	}
	telegramStarsAPI := telegramstars.NewWithOptions(rootCtx, db, repositoryOptions)
	tonAPI := ton.NewWithOptions(rootCtx, db, repositoryOptions)
	plategaAPI := platega.NewWithOptions(rootCtx, db, repositoryOptions)
	vkmaAPI := vkma.NewWithOptions(rootCtx, db, repositoryOptions)
	yooKassaAPI := yookassa.NewWithOptions(rootCtx, db, repositoryOptions)
	assetAPI := asset.NewWithOptions(rootCtx, db, repositoryOptions)
	productAPI := product.NewWithOptions(rootCtx, db, repositoryOptions)
	checkoutAPI := checkout.NewWithOptions(rootCtx, db, repositoryOptions)
	subscriptionAPI := subscription.NewWithOptions(rootCtx, db, repositoryOptions)
	refundAPI := refund.NewWithOptions(rootCtx, db, refundProviders(telegramStarsAPI, tonAPI, plategaAPI, yooKassaAPI), repositoryOptions)
	payments := &Payment{
		Admin:        admin.NewWithServices(rootCtx, db, repositoryOptions, assetAPI, productAPI, checkoutAPI, refundAPI),
		User:         user.New(assetAPI, productAPI, checkoutAPI, subscriptionAPI),
		asset:        assetAPI,
		product:      productAPI,
		checkout:     checkoutAPI,
		refund:       refundAPI,
		subscription: subscriptionAPI,
		Adapters: &Adapters{
			TON:           tonAPI,
			TelegramStars: telegramStarsAPI,
			Platega:       plategaAPI,
			VKMA:          vkmaAPI,
			YooKassa:      yooKassaAPI,
		},
		client:            db,
		ownsClient:        ownsClient,
		callbacks:         callbackutil.NewWithTable(db.DB(), callbackutil.PaymentTable),
		rootCtx:           rootCtx,
		rootCancel:        rootCancel,
		pricing:           repository.NewPaymentRepositoryWithOptions(db, repositoryOptions),
		pricingHTTPClient: options.PriceUpdateHTTPClient,
		pricingInterval:   options.PriceUpdateInterval,
		pricingBaseURL:    options.PriceUpdateBaseURL,
	}
	payments.startPriceUpdater()
	return payments
}

func (a *Payment) Close() error {
	if a == nil {
		return nil
	}
	if a.rootCancel != nil {
		a.rootCancel()
	}
	var err error
	if a.Adapters != nil {
		if a.Adapters.TON != nil {
			err = errors.Join(err, a.Adapters.TON.Close())
		}
		if a.Adapters.TelegramStars != nil {
			err = errors.Join(err, a.Adapters.TelegramStars.Close())
		}
		if a.Adapters.Platega != nil {
			err = errors.Join(err, a.Adapters.Platega.Close())
		}
		if a.Adapters.VKMA != nil {
			err = errors.Join(err, a.Adapters.VKMA.Close())
		}
		if a.Adapters.YooKassa != nil {
			err = errors.Join(err, a.Adapters.YooKassa.Close())
		}
	}
	a.background.Wait()
	if a.pricingDone != nil {
		<-a.pricingDone
	}
	if a.pricing != nil {
		err = errors.Join(err, a.pricing.Close())
	}
	if a.product != nil {
		err = errors.Join(err, a.product.Close())
	}
	if a.Admin != nil {
		err = errors.Join(err, a.Admin.Close())
	}
	if a.asset != nil {
		err = errors.Join(err, a.asset.Close())
	}
	if a.checkout != nil {
		err = errors.Join(err, a.checkout.Close())
	}
	if a.refund != nil {
		err = errors.Join(err, a.refund.Close())
	}
	if a.subscription != nil {
		err = errors.Join(err, a.subscription.Close())
	}
	if a.callbacks != nil {
		err = errors.Join(err, a.callbacks.Close())
	}
	if a.ownsClient && a.client != nil {
		err = errors.Join(err, a.client.Close())
	}
	return err
}

// IsReady reports whether the service is initialized and its lifecycle is active.
func (a *Payment) IsReady() bool {
	if a == nil {
		return false
	}
	a.lifecycleMu.Lock()
	defer a.lifecycleMu.Unlock()
	return a.rootCtx != nil && a.rootCtx.Err() == nil &&
		a.Admin != nil && a.User != nil && a.Adapters != nil
}

func (a *Payment) bindContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if a == nil {
		return mergeContexts(context.Background(), ctx)
	}
	return mergeContexts(a.rootCtx, ctx)
}

func refundProviders(
	telegramStarsAPI *telegramstars.TelegramStars,
	tonAPI *ton.TON,
	plategaAPI *platega.Platega,
	yooKassaAPI *yookassa.YooKassa,
) map[string]refund.ProviderRefundFunc {
	return map[string]refund.ProviderRefundFunc{
		telegramstars.ProviderCode: func(ctx context.Context, params refund.ProviderRefundParams) (refund.ProviderRefundResult, error) {
			credentials, ok := params.ProviderParams.(telegramstars.Credentials)
			if !ok {
				return refund.ProviderRefundResult{}, ErrTelegramStarsRefundCredentialsRequired
			}
			if params.AmountMinor != params.Attempt.AmountMinor {
				return refund.ProviderRefundResult{}, ErrTelegramStarsFullRefundOnly
			}
			if params.Attempt.ProviderChargeID == nil || *params.Attempt.ProviderChargeID == "" {
				return refund.ProviderRefundResult{}, ErrTelegramStarsChargeIDRequired
			}
			userID, err := strconv.ParseInt(params.Order.PlatformUserID, 10, 64)
			if err != nil {
				return refund.ProviderRefundResult{}, serviceerrors.Wrap(serviceerrors.CodeInvalidFields, ErrTelegramStarsPlatformUserIDInvalid.Message(), err)
			}
			result, err := telegramStarsAPI.Execute(ctx, telegramstars.RefundParams{
				Credentials:             credentials,
				UserID:                  userID,
				TelegramPaymentChargeID: *params.Attempt.ProviderChargeID,
			})
			if err != nil {
				return refund.ProviderRefundResult{}, err
			}
			return refund.ProviderRefundResult{
				ProviderRefundID: result.ProviderRefundID,
				Status:           result.Status,
			}, nil
		},
		yookassa.ProviderCode: func(ctx context.Context, params refund.ProviderRefundParams) (refund.ProviderRefundResult, error) {
			credentials, ok := params.ProviderParams.(yookassa.Credentials)
			if !ok {
				return refund.ProviderRefundResult{}, ErrYooKassaRefundCredentialsRequired
			}
			if params.Attempt.ProviderPaymentID == nil || *params.Attempt.ProviderPaymentID == "" {
				return refund.ProviderRefundResult{}, ErrYooKassaPaymentIDRequired
			}
			result, err := yooKassaAPI.Execute(ctx, yookassa.RefundParams{
				Credentials:    credentials,
				PaymentID:      *params.Attempt.ProviderPaymentID,
				AmountMinor:    params.AmountMinor,
				AssetCode:      params.Attempt.AssetCode,
				Description:    params.Reason,
				IdempotencyKey: fmt.Sprintf("payment-refund-%d", params.RefundID),
			})
			if err != nil {
				return refund.ProviderRefundResult{}, err
			}
			return refund.ProviderRefundResult{
				ProviderRefundID: result.ProviderRefundID,
				Status:           result.Status,
			}, nil
		},
		platega.ProviderCode: func(ctx context.Context, params refund.ProviderRefundParams) (refund.ProviderRefundResult, error) {
			providerParams, ok := params.ProviderParams.(platega.RefundParams)
			if !ok {
				return refund.ProviderRefundResult{}, ErrPlategaRefundParamsRequired
			}
			if params.Attempt.ProviderPaymentID != nil {
				providerParams.TransactionID = *params.Attempt.ProviderPaymentID
			}
			providerParams.AmountMinor = params.AmountMinor
			providerParams.AssetCode = params.Attempt.AssetCode
			providerParams.Reason = params.Reason
			providerParams.IdempotencyKey = fmt.Sprintf("payment-refund-%d", params.RefundID)
			result, err := plategaAPI.Execute(ctx, platega.RefundParams{
				Executor:       providerParams.Executor,
				TransactionID:  providerParams.TransactionID,
				AmountMinor:    providerParams.AmountMinor,
				AssetCode:      providerParams.AssetCode,
				Reason:         providerParams.Reason,
				IdempotencyKey: providerParams.IdempotencyKey,
			})
			return refund.ProviderRefundResult{ProviderRefundID: result.ProviderRefundID, Status: result.Status}, err
		},
		ton.ProviderCode: func(ctx context.Context, params refund.ProviderRefundParams) (refund.ProviderRefundResult, error) {
			providerParams, ok := params.ProviderParams.(ton.RefundParams)
			if !ok {
				return refund.ProviderRefundResult{}, ErrTONRefundParamsRequired
			}
			providerParams.AssetCode = params.Attempt.AssetCode
			providerParams.AmountMinor = params.AmountMinor
			providerParams.Comment = params.Reason
			providerParams.IdempotencyKey = fmt.Sprintf("payment-refund-%d", params.RefundID)
			result, err := tonAPI.Execute(ctx, providerParams)
			return refund.ProviderRefundResult{ProviderRefundID: result.ProviderRefundID, Status: result.Status}, err
		},
	}
}
