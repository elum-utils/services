package subscription

type IsActiveParams struct {
	WorkspaceID    string
	PlatformID     int64
	PlatformUserID string
	ProductID      string
	ProviderCode   string
}
