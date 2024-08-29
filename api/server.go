package api

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/Interlocked-Labs/oracle-provider/logging"
	"github.com/Interlocked-Labs/oracle-provider/modules/tokentracker"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Server struct {
	*http.Server
}

var log *logrus.Entry

func init() {
	log = logging.NewLogger().WithField("module", "server")
}

func NewServer(configPath string) (*Server, error) {
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	log.Info("Starting CRON")

	db_path := viper.GetString("DB_CONFIG")
	providers := viper.GetStringSlice("Provider")
	twapMaxTimeDeltaSeconds := viper.GetInt64("TwapMaxTimeDeltaSeconds")
	twapMinHistoryPeriodFraction := viper.GetFloat64("TwapMinHistoryPeriodFraction")
	twapMaxPriceDeviation := viper.GetFloat64("TwapMaxPriceDeviation")
	twapPeriodInMinute := viper.GetDuration("TwapPeriodInMinute")
	token_config, _ := tokentracker.GetConfig("./token_config.json")
	expiryInSeconds := viper.GetInt64("EXPIRY_IN_SECONDS")

	oracle := tokentracker.New(db_path, providers, twapMaxTimeDeltaSeconds, twapMinHistoryPeriodFraction, twapMaxPriceDeviation, twapPeriodInMinute, token_config, expiryInSeconds)
	api, err := New(viper.GetBool("enable_cors"), oracle)
	if err != nil {
		return nil, err
	}
	cronFetchInterval, cronSubmitInterval := getCronInterval()
	c := cron.New()
	log.WithFields(logrus.Fields{"cronInterval": cronFetchInterval}).Info("Start CRON for every")
	c.Schedule(cron.Every(cronFetchInterval), createFetchCronJob(oracle))
	c.Start()

	d := cron.New()
	log.WithFields(logrus.Fields{"cronInterval": cronSubmitInterval}).Info("Start CRON for every")
	d.Schedule(cron.Every(cronSubmitInterval), createSubmitCronJob(oracle))
	d.Start()

	addr := getServerAddress()

	srv := http.Server{
		Addr:    addr,
		Handler: api,
	}

	return &Server{&srv}, nil
}

func getCronInterval() (time.Duration, time.Duration) {
	cronFetchInterval := viper.GetDuration("CRON_INTERVAL_IN_SEC_FETCH")
	if cronFetchInterval < 1 {
		cronFetchInterval = 300
	}

	cronSubmitInterval := viper.GetDuration("CRON_INTERVAL_IN_SEC_SUBMIT")
	if cronSubmitInterval < 1 {
		cronSubmitInterval = 600
	}
	return cronFetchInterval * time.Second, cronSubmitInterval * time.Second
}

func calculateAvgFetchTokenPrice(o *tokentracker.Oracle) (interface{}, error) {
	res, err := o.CalcNFetchTokenPrices()
	return res, err
}

func calculateAvgSubmitTokenPrice(o *tokentracker.Oracle) (interface{}, error) {
	res, err := o.CalcNSubmitTokenPrices()
	return res, err
}

func createFetchCronJob(o *tokentracker.Oracle) cron.FuncJob {
	return func() {
		executeCronJob("Fetch Token Price", calculateAvgFetchTokenPrice, o)
	}
}

func createSubmitCronJob(o *tokentracker.Oracle) cron.FuncJob {
	return func() {
		executeCronJob("Submit Token Price", calculateAvgSubmitTokenPrice, o)
	}
}

func executeCronJob(jobName string, fn func(o *tokentracker.Oracle) (interface{}, error), o *tokentracker.Oracle) {
	log.Infof("Running CRON job for %s", jobName)
	result, err := fn(o)
	if err != nil {
		log.WithFields(logrus.Fields{"err": err}).Errorf("Error submitting %s", jobName)
	} else {
		log.WithFields(logrus.Fields{jobName: result}).Infof("%s executed", jobName)
	}
}

func getServerAddress() string {
	port := viper.GetString("PORT")
	if port == "" {
		port = ":3001"
	}

	if strings.Contains(port, ":") {
		return port
	}
	return "localhost:" + port
}

func (srv *Server) Start() {
	log.Info("Starting server")
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			panic(err)
		}
	}()
	log.Info("Listening on: ", srv.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	sig := <-quit
	log.Info("Shutting down server... Reason:", sig)

	if err := srv.Shutdown(context.Background()); err != nil {
		panic(err)
	}
	log.Info("Server gracefully stopped")
}
