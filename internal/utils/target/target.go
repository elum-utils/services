package target

import (
	"strconv"
	"strings"

	json "github.com/goccy/go-json"
)

type Context struct {
	IsPremium  bool
	Sex        string
	Country    string
	Locale     string
	Platform   string
	PlatformID int64
}

type Rules struct {
	IsPremium   *bool    `json:"is_premium"`
	Sex         []string `json:"sex"`
	Country     []string `json:"country"`
	Countries   []string `json:"countries"`
	Loc         []string `json:"loc"`
	Locale      []string `json:"locale"`
	Locales     []string `json:"locales"`
	Platform    []string `json:"platform"`
	PlatformID  []string `json:"platform_id"`
	PlatformIDs []string `json:"platform_ids"`
}

func Match(raw json.RawMessage, ctx Context) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return true
	}
	var rules Rules
	if err := json.Unmarshal(raw, &rules); err != nil {
		return false
	}
	if rules.IsPremium != nil && *rules.IsPremium != ctx.IsPremium {
		return false
	}
	if len(rules.Sex) > 0 && !containsFold(rules.Sex, ctx.Sex) {
		return false
	}
	countries := append(append([]string{}, rules.Country...), rules.Countries...)
	if len(countries) > 0 && !containsFold(countries, ctx.Country) {
		return false
	}
	locales := append(append(append([]string{}, rules.Loc...), rules.Locale...), rules.Locales...)
	if len(locales) > 0 && !containsFold(locales, ctx.Locale) {
		return false
	}
	if len(rules.Platform) > 0 && !matchesPlatform(rules.Platform, ctx) {
		return false
	}
	platformIDs := append(append([]string{}, rules.PlatformID...), rules.PlatformIDs...)
	if len(platformIDs) > 0 && !matchesPlatformID(platformIDs, ctx.PlatformID) {
		return false
	}
	return true
}

func (r *Rules) UnmarshalJSON(data []byte) error {
	type rawRules struct {
		IsPremium   *bool           `json:"is_premium"`
		Sex         json.RawMessage `json:"sex"`
		Country     json.RawMessage `json:"country"`
		Countries   json.RawMessage `json:"countries"`
		Loc         json.RawMessage `json:"loc"`
		Locale      json.RawMessage `json:"locale"`
		Locales     json.RawMessage `json:"locales"`
		Platform    json.RawMessage `json:"platform"`
		PlatformID  json.RawMessage `json:"platform_id"`
		PlatformIDs json.RawMessage `json:"platform_ids"`
	}
	var raw rawRules
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.IsPremium = raw.IsPremium
	r.Sex = stringList(raw.Sex)
	r.Country = stringList(raw.Country)
	r.Countries = stringList(raw.Countries)
	r.Loc = stringList(raw.Loc)
	r.Locale = stringList(raw.Locale)
	r.Locales = stringList(raw.Locales)
	r.Platform = stringList(raw.Platform)
	r.PlatformID = stringList(raw.PlatformID)
	r.PlatformIDs = stringList(raw.PlatformIDs)
	return nil
}

func stringList(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		if single == "" {
			return nil
		}
		return []string{single}
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err == nil {
		return []string{number.String()}
	}
	var rawList []json.RawMessage
	if err := json.Unmarshal(raw, &rawList); err != nil {
		return nil
	}
	out := make([]string, 0, len(rawList))
	for _, item := range rawList {
		var value string
		if err := json.Unmarshal(item, &value); err != nil {
			var number json.Number
			if err := json.Unmarshal(item, &number); err == nil {
				value = number.String()
			}
		}
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func containsFold(values []string, target string) bool {
	if target == "" {
		return false
	}
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func matchesPlatform(values []string, ctx Context) bool {
	if ctx.Platform != "" && containsFold(values, ctx.Platform) {
		return true
	}
	if matchesPlatformID(values, ctx.PlatformID) {
		return true
	}
	return false
}

func matchesPlatformID(values []string, platformID int64) bool {
	if platformID != 0 && containsFold(values, strconv.FormatInt(platformID, 10)) {
		return true
	}
	return false
}
