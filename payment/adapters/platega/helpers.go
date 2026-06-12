package platega

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	utils "github.com/elum-utils/services/internal/utils"
	"github.com/go-sql-driver/mysql"
)

func rubMajorFromMinor(amountMinor uint64) float64 {
	return float64(amountMinor) / 100
}

func rubMinorFromMajor(amount float64) uint64 {
	return uint64(math.Round(amount * 100))
}

func normalizeLocale(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if locale == "" {
		return "ru"
	}
	return locale
}

func validateHeaders(headers http.Header, credentials Credentials) bool {
	if credentials.MerchantID == "" || credentials.Secret == "" {
		return false
	}
	return constantTimeString(headers.Get("X-MerchantId"), credentials.MerchantID) &&
		constantTimeString(headers.Get("X-Secret"), credentials.Secret)
}

func constantTimeString(left string, right string) bool {
	if len(left) != len(right) {
		return false
	}
	var diff byte
	for i := 0; i < len(left); i++ {
		diff |= left[i] ^ right[i]
	}
	return diff == 0
}

func webhookEventID(payload callbackPayload) string {
	return fmt.Sprintf("%s:%s:%d", payload.ID, payload.Status, payload.PaymentMethod)
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func nullInt64FromPtr(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func nilIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return utils.Ref(value)
}

func isDuplicateEntry(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}
