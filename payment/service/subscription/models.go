package subscription

import services "github.com/elum-utils/services"

type IsActiveParams struct {
	Identity     services.Identity
	ProductID    string
	ProviderCode string
}
