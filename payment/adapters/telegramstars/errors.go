package telegramstars

import (
	"errors"
	"fmt"

	serviceerrors "github.com/elum-utils/services/errors"
)

var (
	ErrNotInitialized                  = serviceerrors.New(serviceerrors.CodeNotReady, "telegram_stars adapter is not initialized")
	ErrBotTokenRequired                = serviceerrors.New(serviceerrors.CodeInvalidFields, "telegram_stars bot token is required")
	ErrInvoicePayloadRequired          = serviceerrors.New(serviceerrors.CodeInvalidFields, "telegram_stars invoice payload is required")
	ErrTelegramPaymentChargeIDRequired = serviceerrors.New(serviceerrors.CodeInvalidFields, "telegram_stars payment charge id is required")
)

func wrapAPIError(action string, status int, code int, description string, body string) error {
	return serviceerrors.Wrap(
		serviceerrors.CodeUnavailable,
		fmt.Sprintf("telegram_stars %s failed with status %d code %d", action, status, code),
		errors.New(description+": "+body),
	)
}
