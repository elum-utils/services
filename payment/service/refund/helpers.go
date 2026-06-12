package refund

import utils "github.com/elum-utils/services/internal/utils"

func refIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return utils.Ref(value)
}
