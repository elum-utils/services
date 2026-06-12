package yookassa

import (
	"context"
	"fmt"
	"net/http"
)

func (c *Client) GetPayment(ctx context.Context, paymentID string) (paymentAPIResponse, error) {
	if err := c.requireCredentials(); err != nil {
		return paymentAPIResponse{}, err
	}

	var result paymentAPIResponse
	resp, err := c.rest.R().
		SetContext(ctx).
		SetResult(&result).
		Get("/v3/payments/" + paymentID)
	if err != nil {
		return paymentAPIResponse{}, err
	}
	if resp.StatusCode() < http.StatusOK || resp.StatusCode() >= http.StatusMultipleChoices {
		return paymentAPIResponse{}, fmt.Errorf("yookassa: get payment failed: status=%d body=%s", resp.StatusCode(), resp.String())
	}
	return result, nil
}
