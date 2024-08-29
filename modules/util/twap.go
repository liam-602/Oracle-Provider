package util

import (
	"fmt"
	"sort"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/Interlocked-Labs/oracle-provider/modules/db"
	types "github.com/Interlocked-Labs/oracle-provider/modules/type"
	"github.com/sirupsen/logrus"
)

var (
	TwapMaxTimeDeltaSeconds      = int64(300)
	TwapMinHistoryPeriodFraction = 0.8
	TwapMaxPriceDeviation        = 0.05
	TwapPeriodInMinute           = time.Duration(20)
)

func InitConfig(twapMaxTimeDeltaSeconds int64, twapMinHistoryPeriodFraction float64, twapMaxPriceDeviation float64, twapPeriodInMinute time.Duration) {
	TwapMaxTimeDeltaSeconds = twapMaxTimeDeltaSeconds
	TwapMinHistoryPeriodFraction = twapMinHistoryPeriodFraction
	TwapMaxPriceDeviation = twapMaxPriceDeviation
	TwapPeriodInMinute = twapPeriodInMinute
}

// GetPrices returns the map containing the TWAP prices computed for available providers
func GetPrices(symbol string, p db.PriceHistory) (map[string]types.TickerPrice, error) {
	now := time.Now()

	period := TwapPeriodInMinute * time.Minute
	start := now.Add(-period)
	tickers, err := p.GetTickerPrices(symbol, start, now)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"symbol": symbol,
			"error":  err,
		}).Error("failed to get historical tickers")
		return nil, err
	}

	derivativePrices := map[string]types.TickerPrice{}
	for providerName, tickerPrices := range tickers {
		price, missing, err := Twap(tickerPrices, start, now)
		if err != nil || price.IsNil() || price.IsZero() {
			logrus.WithFields(logrus.Fields{
				"symbol":   symbol,
				"error":    err,
				"provider": providerName,
				"period":   period.String(),
				"missing":  (time.Second * time.Duration(missing)).String(),
			}).Warn("failed to compute twap")
			continue
		}

		derivativePrices[providerName] = types.TickerPrice{
			Price: price,
			Time:  now,
		}
	}
	if len(derivativePrices) == 0 {
		return nil, fmt.Errorf("there are no token prices")
	}
	return derivativePrices, nil
}

// Twap calculates the Time-Weighted Average Price over a specified period.
func Twap(
	tickers []types.TickerPrice,
	start time.Time,
	end time.Time,
) (sdkmath.LegacyDec, int64, error) {
	filteredTickers := []types.TickerPrice{}
	for _, ticker := range tickers {
		if ticker.Time.Before(start) || ticker.Time.After(end) {
			continue
		}

		filteredTickers = append(filteredTickers, ticker)
	}

	// Calculate the median price from the filtered tickers.
	median, err := weightedMedian(tickers)
	if err != nil {
		return sdkmath.LegacyDec{}, 0, err
	}

	if median.IsZero() {
		return sdkmath.LegacyDec{}, 0, fmt.Errorf("median is 0")
	}

	priceTotal := sdkmath.LegacyZeroDec()
	validTimeTotal := int64(0)
	discardedTime := int64(0)

	// Iterate through the filtered tickers to accumulate valid, time-weighted prices.
	for i := 0; i < len(filteredTickers)-1; i++ {
		ticker := filteredTickers[i]
		nextTicker := filteredTickers[i+1]

		// Calculate the time difference to the next ticker.
		timeDelta := nextTicker.Time.Unix() - ticker.Time.Unix()

		if timeDelta > TwapMaxTimeDeltaSeconds {
			discardedTime = discardedTime + timeDelta
			continue
		}

		// Skip large time gaps or significant price deviations.
		if isTickerValid(ticker, median) {
			priceTotal = priceTotal.Add(ticker.Price.MulInt64(timeDelta))
			validTimeTotal += timeDelta
		} else {
			discardedTime += timeDelta // Assuming discardedTime is initialized and managed globally or externally.
		}
	}

	period := end.Sub(start).Seconds()
	minPeriod := int64(TwapMinHistoryPeriodFraction * period)

	// Validate accumulated time against minimum required period.
	if validTimeTotal < minPeriod {
		return sdkmath.LegacyDec{}, discardedTime, fmt.Errorf("too much time gap in history")
	}

	return priceTotal.QuoInt64(validTimeTotal), discardedTime, nil
}

func weightedMedian(tickers []types.TickerPrice) (sdkmath.LegacyDec, error) {
	type Price struct {
		Value  sdkmath.LegacyDec
		Weight int64
	}

	if len(tickers) < 2 {
		return sdkmath.LegacyZeroDec(), nil
	}

	prices := []Price{}

	// Calculate weights and populate prices slice.
	for i := 1; i < len(tickers); i++ {
		weight := tickers[i].Time.Unix() - tickers[i-1].Time.Unix()
		prices = append(prices, Price{
			Value:  tickers[i].Price,
			Weight: weight,
		})
	}

	// Sort prices by value.
	sort.Slice(prices, func(i, j int) bool {
		return prices[i].Value.LT(prices[j].Value)
	})

	totalWeight := tickers[len(tickers)-1].Time.Unix() - tickers[0].Time.Unix()
	cumulativeWeight := int64(0)

	// Find the weighted median.
	for _, price := range prices {
		cumulativeWeight += price.Weight
		if cumulativeWeight > totalWeight/2 {
			return price.Value, nil
		}
	}
	return sdkmath.LegacyZeroDec(), nil
}

// isTickerValid checks if the ticker deviates too significantly from the median price
func isTickerValid(ticker types.TickerPrice, median sdkmath.LegacyDec) bool {
	maxDeviationRatio := sdkmath.LegacyMustNewDecFromStr(fmt.Sprintf("%f", TwapMaxPriceDeviation))
	max := median.Add(median.Mul(maxDeviationRatio))
	min := median.Sub(median.Mul(maxDeviationRatio))

	return ticker.Price.LT(max) && ticker.Price.GT(min)
}
