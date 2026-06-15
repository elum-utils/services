package platega

import (
	"errors"
	"fmt"

	serviceerrors "github.com/elum-utils/services/errors"
)

var (
	ErrNotInitialized            = serviceerrors.New(serviceerrors.CodeNotReady, "platega adapter is not initialized")
	ErrCredentialsRequired       = serviceerrors.New(serviceerrors.CodeInvalidFields, "platega merchant id and secret are required")
	ErrWebhookCredentialsInvalid = serviceerrors.New(serviceerrors.CodeUnauthorized, "platega callback credentials are invalid")
	ErrTransactionIDRequired     = serviceerrors.New(serviceerrors.CodeInvalidFields, "platega transaction id is required")
	ErrTransactionResponseEmpty  = serviceerrors.New(serviceerrors.CodeInternalError, "platega create transaction response has empty transaction id")
	ErrRefundUnsupported         = serviceerrors.New(serviceerrors.CodeUnsupported, "platega refund API is not configured")
)

func wrapAPIError(action string, status int, body string) error {
	return serviceerrors.Wrap(serviceerrors.CodeUnavailable, fmt.Sprintf("platega %s failed with status %d", action, status), errors.New(body))
}
