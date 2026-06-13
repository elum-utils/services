package user

import (
	"context"
	"errors"
)

func (u *User) IsSubscriptionActive(ctx context.Context, params IsSubscriptionActiveParams) (bool, error) {
	if u == nil || u.subscription == nil {
		return false, errors.New("payment user: subscription service is not initialized")
	}
	return u.subscription.IsActive(ctx, params)
}
