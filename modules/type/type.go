package types

import (
	"fmt"
	"time"

	sdkmath "cosmossdk.io/math"
)

type TickerPrice struct {
	Symbol string            `protobuf:"bytes,1,opt,name=symbol,proto3" json:"symbol"`
	Time   time.Time         `protobuf:"bytes,2,opt,name=time,proto3,stdtime" json:"time"`
	Price  sdkmath.LegacyDec `protobuf:"bytes,3,opt,name=price,proto3,customtype=cosmossdk.io/math.LegacyDec" json:"price"`
}

type TokenPrices map[string]float64

func NewTickerPrice(symbol string, time time.Time, price string) (TickerPrice, error) {
	priceDec, err := sdkmath.LegacyNewDecFromStr(price)
	if err != nil {
		return TickerPrice{}, fmt.Errorf("failed to convert ticker price: %v", err)
	}
	ticker := TickerPrice{
		Symbol: symbol,
		Time:   time,
		Price:  priceDec,
	}
	return ticker, nil
}
