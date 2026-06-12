package ton

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
)

func normalizeLocale(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if locale == "" {
		return "ru"
	}
	return locale
}

func normalizeNetwork(network string) string {
	network = strings.ToLower(strings.TrimSpace(network))
	if network == "" {
		return NetworkMainnet
	}
	return network
}

func validateNetwork(network string) (string, error) {
	network = normalizeNetwork(network)
	switch network {
	case NetworkMainnet, NetworkTestnet:
		return network, nil
	default:
		return "", errors.New("ton: network must be ton.NetworkMainnet or ton.NetworkTestnet")
	}
}

func nullInt64FromPtr(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
}

func nullInt64FromUint64(value uint64) sql.NullInt64 {
	if value == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(value), Valid: true}
}

func uint64FromNull(value sql.NullInt64) uint64 {
	if !value.Valid || value.Int64 <= 0 {
		return 0
	}
	return uint64(value.Int64)
}

func isDuplicateEntry(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}
