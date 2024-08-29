package tokentracker

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Interlocked-Labs/oracle-provider/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/sirupsen/logrus"
)

type Resource struct {
	Oracle *Oracle
}

var (
	log *logrus.Entry
)

func init() {
	loggers := logging.NewLogger()
	log = loggers.WithField("module", "tokentracker")
}

func NewResource(o *Oracle) (*Resource, error) {
	resource := &Resource{Oracle: o}
	return resource, nil
}

func (rs *Resource) Router() *chi.Mux {
	r := chi.NewRouter()
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Get("/", rs.GetTokenPricePerChain)
	return r
}

func (rs *Resource) GetTokenPricePerChain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Content-Type", "application/json")
	response, err := rs.Oracle.CalcNSubmitTokenPrices()
	if err != nil {
		fmt.Println("error while submitting token prices: ", err)
	}
	rss, err := json.Marshal(response)
	if err != nil {
		fmt.Println("error while marshalling: ", err)
	}
	_, err = w.Write([]byte(rss))
	if err != nil {
		fmt.Println("error while writing: ", err)
	}
}
