package tokentracker

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/Interlocked-Labs/oracle-provider/logging"
	"github.com/Interlocked-Labs/oracle-provider/modules/db"
	types "github.com/Interlocked-Labs/oracle-provider/modules/type"
	"github.com/Interlocked-Labs/oracle-provider/modules/util"
	oracleTypes "github.com/Interlocked-Labs/sdk-go/frescochain/oracle/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Oracle struct {
	log             *logrus.Entry
	priceHistory    db.PriceHistory
	providers       []string
	token_config    *Config
	expiryInSeconds int64
}

func New(
	db_path string,
	providers []string,
	twapMaxTimeDeltaSeconds int64,
	twapMinHistoryPeriodFraction float64,
	twapMaxPriceDeviation float64,
	twapPeriodInMinute time.Duration,
	token_config *Config,
	expiryInSeconds int64,
) *Oracle {
	loggers := logging.NewLogger()
	util.InitConfig(twapMaxTimeDeltaSeconds, twapMinHistoryPeriodFraction, twapMaxPriceDeviation, twapPeriodInMinute)
	log := loggers.WithField("module", "tokentracker")
	priceHistory, err := db.NewPriceHistory(db_path, log)
	if err != nil {
		log.Printf("Failed to init price history db: %v", err)
	}
	return &Oracle{
		log:             log,
		priceHistory:    priceHistory,
		providers:       providers,
		token_config:    token_config,
		expiryInSeconds: expiryInSeconds,
	}
}

func (o *Oracle) CalcNFetchTokenPrices() (TokenPrices, error) {
	tokenPrices, err := o.fetchTokenPrices()
	if err != nil {
		log.WithFields(logrus.Fields{"err": err}).Error("Error fetching token prices")
		return nil, err
	}

	return tokenPrices, err
}

func (o *Oracle) CalcNSubmitTokenPrices() (TokenPrices, error) {
	tokenPrices, err := o.fetchTokenPrices()
	if err != nil {
		log.WithFields(logrus.Fields{"err": err}).Error("Error fetching token prices")
		return nil, err
	}

	err = o.submitTokenPrices(tokenPrices)
	if err != nil {
		return nil, err
	}
	return tokenPrices, err
}

func (o *Oracle) fetchTokenPrices() (TokenPrices, error) {
	// fetch custom tokens for which price required
	customTokens, err := o.fetchCustomTokens()
	if err != nil {
		log.WithFields(logrus.Fields{"err": err}).Error("Error fetching custom tokens")
		return nil, err
	}

	tokenPricesBySymbol := make(TokenPrices)
	for _, tokens := range customTokens {
		symbol := tokens.Symbol
		log.WithFields(logrus.Fields{"symbol": symbol}).Info("Fetching token price for symbol")
		pricesFromMultipleServices := make([]float64, 0)
		tokenPrices := map[string]types.TickerPrice{}
		t := time.Now()
		//1. Fetch token prices from coingecko for all supported tokens
		for _, provider := range o.providers {
			switch provider {
			case "CoinGecko":
				coinGeckoPrices, err := fetchTokenPricesFromCoingecko(symbol, tokens.CoinGeckoId)
				if coinGeckoPrices == 0 || err != nil {
					log.WithFields(logrus.Fields{"err": err, "coinGeckoPrices": coinGeckoPrices}).Error("Error fetching token prices from CoinGecko")
				} else {
					tokenPrices[provider] = types.TickerPrice{
						Symbol: symbol,
						Time:   t,
						Price:  sdkmath.LegacyMustNewDecFromStr(fmt.Sprintf("%f", coinGeckoPrices)),
					}
					pricesFromMultipleServices = append(pricesFromMultipleServices, coinGeckoPrices)
					log.WithFields(logrus.Fields{"coinGeckoPrices": coinGeckoPrices}).Info("Fetching token prices from CoinGecko")
					coinGeckoPrices = 0
				}
			case "CoinMarketCap":
				coinCMCPrices, err := fetchTokenPricesFromCoinMarketCap(symbol)
				if coinCMCPrices == 0 || err != nil {
					log.WithFields(logrus.Fields{"err": err, "coinCMCPrices": coinCMCPrices}).Error("Error fetching token prices from CoinMarketCap")
				} else {
					tokenPrices[provider] = types.TickerPrice{
						Symbol: symbol,
						Time:   t,
						Price:  sdkmath.LegacyMustNewDecFromStr(fmt.Sprintf("%f", coinCMCPrices)),
					}
					pricesFromMultipleServices = append(pricesFromMultipleServices, coinCMCPrices)
					log.WithFields(logrus.Fields{"coinCMCPrices": coinCMCPrices}).Info("Fetching token prices from CoinMarketCap")
					coinCMCPrices = 0
				}
			case "CoinAPI":
				coinAPIPrices, err := fetchTokenPricesFromCoinAPI(symbol)
				if coinAPIPrices == 0 || err != nil {
					log.WithFields(logrus.Fields{"err": err, "coinAPIPrices": coinAPIPrices}).Error("Error fetching token prices from CoinAPI")
				} else {
					tokenPrices[provider] = types.TickerPrice{
						Symbol: symbol,
						Time:   t,
						Price:  sdkmath.LegacyMustNewDecFromStr(fmt.Sprintf("%f", coinAPIPrices)),
					}
					pricesFromMultipleServices = append(pricesFromMultipleServices, coinAPIPrices)
					log.WithFields(logrus.Fields{"coinAPIPrices": coinAPIPrices}).Info("Fetching token prices from CoinAPI")
					coinAPIPrices = 0
				}
			default:
				log.WithFields(logrus.Fields{"Provider": provider}).Error("This provider is not defined")
			}
		}
		o.addPrices(tokenPrices)
		medianPrice := o.calculateMedianForTokenPrices(pricesFromMultipleServices)
		if medianPrice == 0 {
			log.WithFields(logrus.Fields{
				"symbol":      symbol,
				"medianPrice": medianPrice,
			}).Error("Error calculating median price")
			continue
		}
		log.WithFields(logrus.Fields{
			"symbol":      symbol,
			"medianPrice": medianPrice,
		}).Info("Median token prices")
		tokenPricesBySymbol[symbol] = medianPrice
	}
	log.WithFields(logrus.Fields{"tokenPricesBySymbol": tokenPricesBySymbol}).Info("Token prices by symbol")
	return tokenPricesBySymbol, nil
}

func (o *Oracle) calculateMedianForTokenPrices(tokenPrices []float64) float64 {
	n := len(tokenPrices)
	if n == 0 {
		return 0
	}
	sort.Float64s(tokenPrices)
	middle := n / 2
	if n%2 == 0 {
		return (tokenPrices[middle-1] + tokenPrices[middle]) / 2
	} else {
		return tokenPrices[middle]
	}
}

func (o *Oracle) submitTokenPrices(tokenPricesBySymbol TokenPrices) error {
	log.WithFields(logrus.Fields{"tokenPricesBySymbol": tokenPricesBySymbol}).Info("Submitting token prices")

	chainClient, err := util.GetChainClientInstance()
	if err != nil {
		log.WithFields(logrus.Fields{"err": err}).Error("Error getting chain client")
		return err
	}

	customTokens, err := o.fetchCustomTokens()
	if err != nil {
		log.WithFields(logrus.Fields{"err": err}).Error("Error fetching custom tokens")
		return err
	}

	expiryInSeconds := viper.GetInt64("EXPIRY_IN_SECONDS")

	tokenPrices := make([]oracleTypes.CurrentPrice, 0)
	//Create tokenPrices msg
	for _, token := range customTokens {
		prices, err := util.GetPrices(token.Symbol, o.priceHistory)
		if err != nil {
			fmt.Println("Error while Getting Prices:", err)
		}
		pricesFromMultipleServices := make([]float64, 0)
		for _, price := range prices {
			decStr := price.Price.String()
			floatVal, err := strconv.ParseFloat(decStr, 64)
			if err != nil {
				fmt.Println("Error parsing float:", err)
			}
			pricesFromMultipleServices = append(pricesFromMultipleServices, floatVal)
		}
		price := o.calculateMedianForTokenPrices(pricesFromMultipleServices)
		priceInDec, err := sdkmath.LegacyNewDecFromStr(fmt.Sprintf("%f", price))
		if err != nil {
			log.Error("Unrecognized token price")
			continue
		}
		log.WithFields(logrus.Fields{"symbol": token.Symbol, "tokenPrice": priceInDec}).Info("Token price")

		tokenPrices = append(tokenPrices, oracleTypes.NewCurrentPrice(fmt.Sprintf("%s:usd", token.Symbol), priceInDec))
	}

	msg := oracleTypes.MsgPostPrice{
		From:        chainClient.FromAddress().String(),
		TokenPrices: tokenPrices,
		Expiry:      time.Now().Add(time.Duration(expiryInSeconds * time.Second.Nanoseconds())),
	}
	log.Info("Broadcasting tokenPrices msg: ", msg)

	_, err = chainClient.SyncBroadcastMsg(&msg)
	if err != nil {
		log.WithFields(logrus.Fields{"msg": msg, "err": err}).Error("Error broadcasting tokenPrices msg")
		return err
	}

	return nil
}

type TokenSymbols struct {
	Symbol         string
	NativeDecimals int64
	CoinGeckoId    string
}

func (o *Oracle) fetchCustomTokens() ([]TokenSymbols, error) {
	var customTokens []TokenSymbols
	for _, token := range o.token_config.CustomTokens {
		customTokens = append(customTokens, TokenSymbols{
			Symbol:         token.Symbol,
			NativeDecimals: int64(token.Decimals),
			CoinGeckoId:    token.CoinGeckoId,
		})
	}
	return customTokens, nil
}

func (o *Oracle) addPrices(tokenPrices map[string]types.TickerPrice) {
	for provider, price := range tokenPrices {
		err := o.priceHistory.AddTickerPrice(price, provider)
		if err != nil {
			log.Errorf("Can't save the %v data: %v", provider, err)
		}
	}
}
