package services

import (
	"testing"

	json "github.com/goccy/go-json"
)

func TestRewardPayloadJSON(t *testing.T) {
	day := "day"
	value := RewardPayload{
		Identity: Identity{
			WorkspaceID: "workspace",
			AppID:       1, PlatformID: 2, PlatformUserID: "3",
		},
		Rewards: []Reward{
			{Key: "coin", Type: "quantity", Quantity: 10, Scale: 2},
			{Key: "premium", Type: "duration", Quantity: 1, Unit: &day},
		},
	}

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	var decoded RewardPayload
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.WorkspaceID != value.WorkspaceID ||
		decoded.AppID != value.AppID ||
		decoded.PlatformID != value.PlatformID ||
		decoded.PlatformUserID != value.PlatformUserID ||
		len(decoded.Rewards) != 2 ||
		decoded.Rewards[0].Scale != 2 ||
		decoded.Rewards[1].Unit == nil ||
		*decoded.Rewards[1].Unit != day {
		t.Fatalf("unexpected decoded payload: %+v", decoded)
	}
}
