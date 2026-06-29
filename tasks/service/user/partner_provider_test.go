package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	json "github.com/goccy/go-json"

	"github.com/elum-utils/services/tasks/repository"
)

func TestTgrassProviderListAndCheck(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/offers", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Auth"); got != "token" {
			t.Fatalf("Auth header = %q", got)
		}
		_, _ = w.Write([]byte(`{
			"status":"ok",
			"offers":[{"name":"Tech","link":"https://t.me/tech","subscribed":false,"type":"channel","channel_id":"-100","offer_id":1054}]
		}`))
	})
	mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"subscribed","is_fake":false}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	secret := "token"
	provider := TgrassProvider{BaseURL: server.URL}
	params := PartnerListProviderParams{
		Identity: Identity{WorkspaceID: "w", PlatformUserID: "123", IsPremium: true},
		Config:   repository.PartnerConfig{Provider: "tgrass", GroupKey: "tgrass", Platform: "telegram", Secret: &secret},
		Locale:   "ru",
		Limit:    1,
	}
	tasks, err := provider.ListPartnerTasks(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].ExternalID != "1054" || tasks[0].ExternalType != "channel" {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
	check, err := provider.CheckPartnerTask(context.Background(), PartnerCheckProviderParams{
		Identity: params.Identity,
		Config:   params.Config,
		Issue: repository.PartnerIssue{
			ExternalID: "1054", PrivatePayload: tasks[0].PrivatePayload,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !check.Completed || check.Status != "subscribed" {
		t.Fatalf("unexpected check: %+v", check)
	}
}

func TestSubGramProviderListAndCheck(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/get-sponsors", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"status":"warning",
			"additional":{"sponsors":[{"ads_id":"42","link":"https://t.me/s","resource_id":"-100","type":"channel","status":"unsubscribed","available_now":true,"button_text":"Join"}]}
		}`))
	})
	mux.HandleFunc("/get-user-subscriptions", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"status":"ok",
			"additional":{"sponsors":[{"link":"https://t.me/s","status":"subscribed"}]}
		}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	secret := "token"
	provider := SubGramProvider{BaseURL: server.URL}
	params := PartnerListProviderParams{
		Identity: Identity{WorkspaceID: "w", PlatformUserID: "123"},
		Config:   repository.PartnerConfig{Provider: "subgram", GroupKey: "subgram", Secret: &secret, Settings: json.RawMessage(`{"action":"task"}`)},
		Locale:   "ru",
	}
	tasks, err := provider.ListPartnerTasks(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].ExternalID != "42:-100" {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
	check, err := provider.CheckPartnerTask(context.Background(), PartnerCheckProviderParams{
		Identity: params.Identity,
		Config:   params.Config,
		Issue:    repository.PartnerIssue{PrivatePayload: tasks[0].PrivatePayload},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !check.Completed || check.Status != "subscribed" {
		t.Fatalf("unexpected check: %+v", check)
	}
}

func TestFlyerProviderListAndCheck(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/get_tasks", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tasks":[{"signature":"sig","task_type":"subscribe channel","link":"https://t.me/c","title":"Channel"}]}`))
	})
	mux.HandleFunc("/check_task", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"completed"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	secret := "key"
	provider := FlyerProvider{BaseURL: server.URL}
	params := PartnerListProviderParams{
		Identity: Identity{WorkspaceID: "w", PlatformUserID: "123"},
		Config:   repository.PartnerConfig{Provider: "flyer", GroupKey: "flyer", Platform: "telegram", Secret: &secret},
		Locale:   "ru",
	}
	tasks, err := provider.ListPartnerTasks(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].ExternalID != "sig" {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
	check, err := provider.CheckPartnerTask(context.Background(), PartnerCheckProviderParams{
		Identity: params.Identity,
		Config:   params.Config,
		Issue:    repository.PartnerIssue{PrivatePayload: tasks[0].PrivatePayload},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !check.Completed || check.Status != "completed" {
		t.Fatalf("unexpected check: %+v", check)
	}
}
