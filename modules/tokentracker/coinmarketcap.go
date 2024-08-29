package tokentracker

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/viper"
)

var (
	coinMarketCapApiKey = viper.GetString("COIN_MARKET_CAP_KEY")
	baseURL             = "https://pro-api.coinmarketcap.com/v1/cryptocurrency"
	coinMarketCapApiUrl = baseURL + "/quotes/latest?symbol="
)

type ApiResponse struct {
	Data map[string]struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
		Quote  map[string]struct {
			Price float64 `json:"price"`
		} `json:"quote"`
	} `json:"data"`
}

func fetchTokenPricesFromCoinMarketCap(symbol string) (float64, error) {
	url := coinMarketCapApiUrl + symbol
	log.Info("Fetching token price from CoinMarketCap for symbol: ", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	if coinMarketCapApiKey == "" {
		coinMarketCapApiKey = viper.GetString("COIN_MARKET_CAP_KEY")
	}
	req.Header.Set("X-CMC_PRO_API_KEY", coinMarketCapApiKey)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil && !strings.Contains(err.Error(), "EOF") {
		return 0, err
	}

	var apiResp ApiResponse
	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return 0, err
	}

	price := apiResp.Data[strings.ToUpper(symbol)].Quote["USD"].Price
	return price, nil
}
