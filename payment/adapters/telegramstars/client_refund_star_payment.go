package telegramstars

import (
	"context"
	"fmt"
	"net/http"
)

func (c *Client) RefundStarPayment(ctx context.Context, payload refundStarPaymentRequest) error {
	if err := c.requireCredentials(); err != nil {
		return err
	}

	var result botAPIResponse[bool]
	resp, err := c.rest.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		SetResult(&result).
		Post(c.methodPath("refundStarPayment"))
	if err != nil {
		return err
	}
	if resp.StatusCode() < http.StatusOK || resp.StatusCode() >= http.StatusMultipleChoices || !result.OK || !result.Result {
		return fmt.Errorf("telegram_stars: refundStarPayment failed: status=%d code=%d description=%s body=%s", resp.StatusCode(), result.ErrorCode, result.Description, resp.String())
	}
	return nil
}
