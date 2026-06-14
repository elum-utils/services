package payment

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/elum-utils/services/payment/repository"
)

func TestFetchDexScreenerPricesBatchesAddressesAndSelectsLiquidity(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/tokens/v1/ton/token-a,token-b" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(`[
			{"baseToken":{"address":"token-a"},"priceUsd":"1.25","liquidity":{"usd":100}},
			{"baseToken":{"address":"token-a"},"priceUsd":"1.30","liquidity":{"usd":500}},
			{"baseToken":{"address":"token-b"},"priceUsd":"0.004","liquidity":{"usd":250}},
			{"baseToken":{"address":"unexpected"},"priceUsd":"999","liquidity":{"usd":999999}}
		]`)),
			Request: r,
		}, nil
	})}

	prices, err := fetchDexScreenerPrices(
		context.Background(),
		client,
		"https://dex.example",
		"ton",
		[]repository.DueAssetRateUpdate{
			{SourceTokenAddress: "token-a"},
			{SourceTokenAddress: "token-b"},
			{SourceTokenAddress: "token-a"},
		},
	)
	if err != nil {
		t.Fatalf("fetch prices: %v", err)
	}
	if prices["token-a"] != 1_300_000 {
		t.Fatalf("unexpected token-a price: %d", prices["token-a"])
	}
	if prices["token-b"] != 4_000 {
		t.Fatalf("unexpected token-b price: %d", prices["token-b"])
	}
	if _, ok := prices["unexpected"]; ok {
		t.Fatal("unexpected token must not be returned")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestSelectDexScreenerPricesRejectsQuoteAndInvalidPrice(t *testing.T) {
	pairs := []dexScreenerPair{{PriceUSD: "0"}}
	pairs[0].BaseToken.Address = "token-a"

	prices := selectDexScreenerPrices(pairs, map[string]struct{}{"token-a": {}})
	if len(prices) != 0 {
		t.Fatalf("expected invalid price to be ignored: %#v", prices)
	}
}

func TestSelectDexScreenerPricesCalculatesQuoteTokenUSDPrice(t *testing.T) {
	pair := dexScreenerPair{
		PriceUSD:    "1.0019",
		PriceNative: "0.5788",
	}
	pair.BaseToken.Address = "usdt"
	pair.QuoteToken.Address = "ton"
	pair.Liquidity = &struct {
		USD float64 `json:"usd"`
	}{USD: 6_525_315}

	prices := selectDexScreenerPrices(
		[]dexScreenerPair{pair},
		map[string]struct{}{"ton": {}},
	)
	if prices["ton"] != 1_730_996 {
		t.Fatalf("unexpected TON price: %d", prices["ton"])
	}
}
