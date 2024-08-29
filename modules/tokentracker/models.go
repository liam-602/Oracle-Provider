package tokentracker

type TokenPrices map[string]float64
type TokenPriceResponse struct {
	ChainId        string
	GasPriceinGwei string
}

type Resource struct {
	Oracle *Oracle
}
