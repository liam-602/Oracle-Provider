package tokentracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type MarketData struct {
	Id  string  `json:"id"`
	USD float64 `json:"current_price"`
}

type Coin struct {
	Id     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

type CoinCache struct {
	coins []Coin
	mu    sync.RWMutex
}

var cache = CoinCache{}

func (c *CoinCache) fetchCoinIds() ([]Coin, error) {
	c.mu.RLock()
	if len(c.coins) > 0 {
		log.Info("Reading from cache")
		defer c.mu.RUnlock()
		return c.coins, nil
	}
	c.mu.RUnlock()

	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Get("https://api.coingecko.com/api/v3/coins/list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response status: %s", resp.Status)
	}

	var coins []Coin
	if err := json.NewDecoder(resp.Body).Decode(&coins); err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.coins = coins
	c.mu.Unlock()

	return coins, nil
}

func filterCoinsBySymbol(coins []Coin, symbol string) Coin {
	for _, coin := range coins {
		if strings.EqualFold(coin.Symbol, symbol) {
			return coin
		}
	}
	return Coin{}
}

func fetchTokenPricesFromCoingecko(symbol string, coingeckoId string) (float64, error) {
	log.Infof("Fetching token price from Coingecko for symbol: %s", symbol)
	if coingeckoId == "" {
		log.Infof("COINGECKO ID NOT FOUND for symbol: %s", symbol)
		coinIds, err := cache.fetchCoinIds()
		if err != nil {
			return 0, err
		}
		coin := filterCoinsBySymbol(coinIds, symbol)
		coingeckoId = coin.Id
	}
	if coingeckoId == "" {
		log.Errorf("COIN ID NOT FOUND for symbol: %s", symbol)
		return 0, errors.New("coin id not found")
	}

	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&ids=%s", coingeckoId)
	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("bad response status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var marketData []MarketData
	if err := json.Unmarshal(body, &marketData); err != nil {
		return 0, err
	}

	if len(marketData) == 0 {
		return 0, errors.New("no market data available")
	}

	return marketData[0].USD, nil
}
