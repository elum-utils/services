package admin

import serviceerrors "github.com/elum-utils/services/errors"

var (
	ErrProductServiceNotInitialized  = serviceerrors.New(serviceerrors.CodeNotReady, "payment admin product service is not initialized")
	ErrAssetServiceNotInitialized    = serviceerrors.New(serviceerrors.CodeNotReady, "payment admin asset service is not initialized")
	ErrCheckoutServiceNotInitialized = serviceerrors.New(serviceerrors.CodeNotReady, "payment admin checkout service is not initialized")
	ErrRefundServiceNotInitialized   = serviceerrors.New(serviceerrors.CodeNotReady, "payment admin refund service is not initialized")
)
