package db

import (
	"database/sql"
	"time"

	types "github.com/Interlocked-Labs/oracle-provider/modules/type"
	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

type (
	PriceHistory struct {
		db      *sql.DB
		insert  *sql.Stmt
		query   *sql.Stmt
		cleanup *sql.Stmt
		logger  *logrus.Entry
	}
)

func NewPriceHistory(path string, log *logrus.Entry) (PriceHistory, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.WithError(err).Error("failed to create db table")
		return PriceHistory{}, err
	}
	p := PriceHistory{
		db:     db,
		logger: log,
	}
	return p, p.Init()
}

func (p *PriceHistory) Init() error {
	_, err := p.db.Exec(`
		CREATE TABLE IF NOT EXISTS crypto_ticker_prices(
        symbol TEXT NOT NULL,
        time INT NOT NULL,
        price TEXT NOT NULL,
		provider TEXT NOT NULL,
        CONSTRAINT id PRIMARY KEY (symbol, provider, time)
    )`)
	if err != nil {
		p.logger.WithError(err).Error("failed to create db table")
		return err
	}

	_, err = p.db.Exec("VACUUM")
	if err != nil {
		p.logger.WithError(err).Error("failed to vacuum database")
		return err
	}

	insert, err := p.db.Prepare(`
		INSERT INTO crypto_ticker_prices(symbol, time, price, provider)
        SELECT ?, ?, ?, ?
        WHERE NOT EXISTS (SELECT 1 FROM crypto_ticker_prices WHERE symbol = ? AND time = ? AND provider = ?)
    `)
	if err != nil {
		p.logger.WithError(err).Error("failed to prepare sql insert statement")
		return err
	}

	query, err := p.db.Prepare(`
		SELECT time, price, provider FROM crypto_ticker_prices
        WHERE symbol = ? AND time BETWEEN ? AND ?
        ORDER BY time ASC
    `)
	if err != nil {
		p.logger.WithError(err).Error("failed to prepare sql query statement")
		return err
	}

	cleanup, err := p.db.Prepare(`
		DELETE from crypto_ticker_prices
		WHERE symbol = ? AND time < ?
	`)
	if err != nil {
		p.logger.WithError(err).Error("failed to prepare sql cleanup statement")
	}

	p.insert = insert
	p.query = query
	p.cleanup = cleanup

	return nil
}

func (p *PriceHistory) AddTickerPrice(ticker types.TickerPrice, provider string) error {
	_, err := p.insert.Exec(
		ticker.Symbol,
		ticker.Time.Unix(),
		ticker.Price.String(),
		provider,
		ticker.Symbol,
		ticker.Time.Unix(),
		provider,
	)
	if err != nil {
		p.logger.WithFields(logrus.Fields{
			"Symbol": ticker.Symbol,
		}).WithError(err).Error("failed to store ticker")
	}
	return err
}

func (p *PriceHistory) GetTickerPrices(
	symbol string,
	start time.Time,
	end time.Time,
) (map[string][]types.TickerPrice, error) {
	logger := p.logger.WithField("symbol", symbol)

	_, err := p.cleanup.Exec(symbol, start.Unix())
	if err != nil {
		logger.WithError(err).Error("failed to remove old ticker prices")
		return nil, err
	}

	rows, err := p.query.Query(symbol, start.Unix(), end.Unix())
	if err != nil {
		logger.WithError(err).Error("failed to query stored ticker prices")
		return nil, err
	}
	defer rows.Close()
	tickers := map[string][]types.TickerPrice{}
	for rows.Next() {
		var epochTime int64
		var price, providerName string
		err := rows.Scan(&epochTime, &price, &providerName)
		if err != nil {
			logger.WithError(err).Error("failed to parse ticker query results")
			return nil, err
		}
		ticker, err := types.NewTickerPrice(symbol, time.Unix(epochTime, 0), price)
		if err != nil {
			logger.WithError(err).Error("failed to create ticker")
		}
		providerTickers, ok := tickers[providerName]
		if !ok {
			tickers[providerName] = []types.TickerPrice{ticker}
		} else {
			tickers[providerName] = append(providerTickers, ticker)
		}
	}
	err = rows.Err()
	if err != nil {
		logger.WithError(err).Error("failed to read all stored tickers")
		return nil, err
	}
	return tickers, nil
}
