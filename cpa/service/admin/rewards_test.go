package admin

import "testing"

func TestValidateReward(t *testing.T) {
	day := "day"
	if rewardType, err := validateReward("coin", "", 1, nil); err != nil || rewardType != "quantity" {
		t.Fatalf("default quantity reward: type=%q err=%v", rewardType, err)
	}
	if rewardType, err := validateReward("premium", "duration", 7, &day); err != nil || rewardType != "duration" {
		t.Fatalf("duration reward: type=%q err=%v", rewardType, err)
	}
	if _, err := validateReward("coin", "quantity", 1, &day); err == nil {
		t.Fatal("quantity reward with duration unit must fail")
	}
}
