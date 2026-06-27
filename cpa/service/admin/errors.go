package admin

import serviceerrors "github.com/elum-utils/services/errors"

var (
	ErrRepositoryNotConfigured       = serviceerrors.New(serviceerrors.CodeNotReady, "cpa admin repository is not configured")
	ErrLocalizationRequired          = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin locale and title are required")
	ErrOfferScopeRequired            = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin workspace and offer id are required")
	ErrOfferPayloadInvalid           = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin payload must be valid JSON")
	ErrOfferRangeInvalid             = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin start_at must be before end_at")
	ErrOfferSharedCodeRequired       = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin shared_code is required")
	ErrOfferCodeSourceRequired       = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin personal code source is required")
	ErrGeneratedCodeLengthInvalid    = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin generated code length must be positive")
	ErrGeneratedCodeAlphabetInvalid  = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin generated alphabet needs at least two symbols")
	ErrPersonalCodeSourceUnsupported = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin unsupported personal code source")
	ErrCodeModeUnsupported           = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin unsupported code mode")
	ErrRewardRequired                = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin reward key and positive quantity are required")
	ErrRewardQuantityUnit            = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin quantity reward must not have duration unit")
	ErrRewardDurationUnit            = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin duration reward requires a valid duration unit")
	ErrRewardTypeUnsupported         = serviceerrors.New(serviceerrors.CodeInvalidFields, "cpa admin reward type must be quantity or duration")
	ErrCodeUploadModeUnsupported     = serviceerrors.New(serviceerrors.CodeFailedPrecondition, "cpa admin codes can only be uploaded to personal pool offers")
)
