package user

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/elum-utils/services/tasks/repository"
)

func TestTgrassProviderLiveManual(t *testing.T) {
	token := os.Getenv("TGRASS_TOKEN")
	userID := os.Getenv("TGRASS_USER_ID")
	if token == "" || userID == "" {
		t.Skip("set TGRASS_TOKEN and TGRASS_USER_ID to run live Tgrass check")
	}
	provider := TgrassProvider{Timeout: 15 * time.Second}
	params := PartnerListProviderParams{
		Identity: Identity{
			WorkspaceID:    "live",
			Platform:       "tma",
			PlatformUserID: userID,
		},
		Config: repository.PartnerConfig{
			WorkspaceID: "live",
			Provider:    "tgrass",
			GroupKey:    "tgrass",
			Platform:    "telegram",
			Secret:      &token,
		},
		Locale: "ru",
		Limit:  1,
		Now:    time.Now().UTC(),
	}
	tasks, err := provider.ListPartnerTasks(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) == 0 {
		offerID := os.Getenv("TGRASS_OFFER_ID")
		if offerID == "" {
			t.Skip("Tgrass returned no available tasks")
		}
		tasks = []PartnerExternalTask{{
			ExternalID:     offerID,
			ExternalType:   "channel",
			PrivatePayload: []byte(`{"offer_id":` + offerID + `}`),
		}}
	}
	t.Logf("external_id=%s external_type=%s public_payload=%s", tasks[0].ExternalID, tasks[0].ExternalType, string(tasks[0].PublicPayload))
	check, err := provider.CheckPartnerTask(context.Background(), PartnerCheckProviderParams{
		Identity: params.Identity,
		Config:   params.Config,
		Issue: repository.PartnerIssue{
			ExternalID:     tasks[0].ExternalID,
			ExternalType:   tasks[0].ExternalType,
			PrivatePayload: tasks[0].PrivatePayload,
		},
		Now: params.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("check completed=%t status=%s payload=%s", check.Completed, check.Status, string(check.Payload))
}
