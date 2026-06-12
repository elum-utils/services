package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"sync"

	callbackutil "github.com/elum-utils/services/internal/utils/callback"
	sqlwrap "github.com/elum-utils/services/internal/utils/sql"

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
	"github.com/go-sql-driver/mysql"
)

type Payment struct {
	Admin        *admin.Admin
	Asset        *asset.Asset
	Product      *product.Product
	Checkout     *checkout.Checkout
	Refund       *refund.Refund
	Subscription *subscription.Subscription

	Adapters *Adapters

	callbacks  *callbackutil.Store
	client     *sqlwrap.Client
	ownsClient bool
	rootCtx    context.Context
	rootCancel context.CancelFunc
	background sync.WaitGroup

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
		return nil, err
	}
	return newAPI(ctx, client, false, options), nil
}

func (a *Payment) Run(ctx context.Context) error {
	if a == nil {
		return errors.New("payment: nil service")
	}
	a.lifecycleMu.Lock()
	if a.running {
		a.lifecycleMu.Unlock()
		return errors.New("payment: service is already running")
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
		return err
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
		return err
	}
}

func open(ctx context.Context, params DatabaseParams) (*Payment, error) {
	if params.User == "" {
		return nil, errors.New("payment: database user is required")
	}
	if params.Database == "" {
		return nil, errors.New("payment: database name is required")
	}
	host := params.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := params.Port
	if port <= 0 {
		port = 3306
	}

	cfg := mysql.Config{
		User:                 params.User,
		Passwd:               params.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", host, port),
		DBName:               params.Database,
		ParseTime:            true,
		AllowNativePasswords: true,
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	client, err := sqlwrap.New(db, toSQLWrapOptions(params.Options))
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return newAPI(ctx, client, true, params.Options), nil
}

func (a *Payment) adopt(running *Payment) {
	a.Admin = running.Admin
	a.Asset = running.Asset
	a.Product = running.Product
	a.Checkout = running.Checkout
	a.Refund = running.Refund
	a.Subscription = running.Subscription
	a.Adapters = running.Adapters
	a.callbacks = running.callbacks
	a.client = running.client
	a.ownsClient = running.ownsClient
	a.rootCtx = running.rootCtx
	a.rootCancel = running.rootCancel
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
	return &Payment{
		Admin:        admin.NewWithOptions(rootCtx, db, repositoryOptions),
		Asset:        asset.NewWithOptions(rootCtx, db, repositoryOptions),
		Product:      product.NewWithOptions(rootCtx, db, repositoryOptions),
		Checkout:     checkout.NewWithOptions(rootCtx, db, repositoryOptions),
		Refund:       refund.NewWithOptions(rootCtx, db, refundProviders(telegramStarsAPI, tonAPI, plategaAPI, yooKassaAPI), repositoryOptions),
		Subscription: subscription.NewWithOptions(rootCtx, db, repositoryOptions),
		Adapters: &Adapters{
			TON:           tonAPI,
			TelegramStars: telegramStarsAPI,
			Platega:       plategaAPI,
			VKMA:          vkmaAPI,
			YooKassa:      yooKassaAPI,
		},
		client:     db,
		ownsClient: ownsClient,
		callbacks:  callbackutil.NewWithTable(db.DB(), callbackutil.PaymentTable),
		rootCtx:    rootCtx,
		rootCancel: rootCancel,
	}
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
	if a.Product != nil {
		err = errors.Join(err, a.Product.Close())
	}
	if a.Admin != nil {
		err = errors.Join(err, a.Admin.Close())
	}
	if a.Asset != nil {
		err = errors.Join(err, a.Asset.Close())
	}
	if a.Checkout != nil {
		err = errors.Join(err, a.Checkout.Close())
	}
	if a.Refund != nil {
		err = errors.Join(err, a.Refund.Close())
	}
	if a.Subscription != nil {
		err = errors.Join(err, a.Subscription.Close())
	}
	if a.callbacks != nil {
		err = errors.Join(err, a.callbacks.Close())
	}
	if a.ownsClient && a.client != nil {
		err = errors.Join(err, a.client.Close())
	}
	return err
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
				return refund.ProviderRefundResult{}, errors.New("payment refund: telegram_stars credentials are required")
			}
			if params.AmountMinor != params.Attempt.AmountMinor {
				return refund.ProviderRefundResult{}, errors.New("payment refund: telegram_stars supports full refunds only")
			}
			if params.Attempt.ProviderChargeID == nil || *params.Attempt.ProviderChargeID == "" {
				return refund.ProviderRefundResult{}, errors.New("payment refund: telegram_stars provider charge id is empty")
			}
			userID, err := strconv.ParseInt(params.Order.PlatformUserID, 10, 64)
			if err != nil {
				return refund.ProviderRefundResult{}, fmt.Errorf("payment refund: telegram_stars platform user id must be int64: %w", err)
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
				return refund.ProviderRefundResult{}, errors.New("payment refund: yookassa credentials are required")
			}
			if params.Attempt.ProviderPaymentID == nil || *params.Attempt.ProviderPaymentID == "" {
				return refund.ProviderRefundResult{}, errors.New("payment refund: yookassa provider payment id is empty")
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
				return refund.ProviderRefundResult{}, errors.New("payment refund: platega refund parameters are required")
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
				return refund.ProviderRefundResult{}, errors.New("payment refund: ton refund parameters are required")
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
