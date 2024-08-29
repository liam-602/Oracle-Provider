package tokentracker

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	apiKey = viper.GetString("COIN_API_KEY")
	apiUrl = "https://rest.coinapi.io/v1/exchangerate/"
)

type ExchangeRate struct {
	Time         string  `json:"time"`
	AssetIDBase  string  `json:"asset_id_base"`
	AssetIDQuote string  `json:"asset_id_quote"`
	Rate         float64 `json:"rate"`
}

func fetchTokenPricesFromCoinAPI(symbol string) (float64, error) {
	apiKey = viper.GetString("COIN_API_KEY")
	url := apiUrl + symbol + "/USD?apikey=" + apiKey

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
		return 0, err
	}
	req.Header.Set("X-CoinAPI-Key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error requesting data: ", err)
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var data ExchangeRate
	if err := json.Unmarshal(body, &data); err != nil && !strings.Contains(err.Error(), "EOF") {
		fmt.Println("error decoding data: ", err)
		return 0, err
	}

	log.WithFields(logrus.Fields{"symbol": symbol, "price": data.Rate}).Info("Data from CoinAPI")
	return data.Rate, err
}
