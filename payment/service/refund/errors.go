package refund

import serviceerrors "github.com/elum-utils/services/errors"

var (
	// ErrProviderUnsupported means the selected payment provider does not implement orchestrated refunds.
	ErrProviderUnsupported = serviceerrors.New(
		serviceerrors.CodeUnsupported,
		"payment refund provider does not support orchestrated refunds",
	)
	// ErrAttemptRequired means a refund cannot be executed without a payment attempt.
	ErrAttemptRequired = serviceerrors.New(serviceerrors.CodeInvalidFields, "payment refund attempt is required")
	// ErrAmountInvalid means the requested refund amount is zero or otherwise invalid.
	ErrAmountInvalid = serviceerrors.New(serviceerrors.CodeInvalidFields, "payment refund amount is invalid")
)
