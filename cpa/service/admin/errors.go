package admin

import serviceerrors "github.com/elum-utils/services/errors"

var (
	ErrRepositoryNotConfigured      = serviceerrors.New(serviceerrors.CodeNotReady, "cpa admin repository is not configured")
	ErrCodeUploadModeUnsupported    = serviceerrors.New(serviceerrors.CodeFailedPrecondition, "cpa admin codes can only be uploaded to personal pool offers")
	ErrCallbackEventIDRequired      = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa callback event id is required")
	ErrCallbackRejectReasonRequired = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa callback reject reason is required")
)
