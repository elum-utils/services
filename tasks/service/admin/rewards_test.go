package admin

import "testing"

func TestValidateReward(t *testing.T) {
	hour := "hour"
	if rewardType, err := validateReward(RewardModel{Key: "coin", Quantity: 1}); err != nil || rewardType != "quantity" {
		t.Fatalf("default quantity reward: type=%q err=%v", rewardType, err)
	}
	if rewardType, err := validateReward(RewardModel{
		Key: "energy", Type: "duration", Quantity: 3, Unit: &hour,
	}); err != nil || rewardType != "duration" {
		t.Fatalf("duration reward: type=%q err=%v", rewardType, err)
	}
	if _, err := validateReward(RewardModel{Key: "energy", Type: "duration", Quantity: 3}); err == nil {
		t.Fatal("duration reward without unit must fail")
	}
}
