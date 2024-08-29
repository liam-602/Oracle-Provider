package tokentracker

import (
	"encoding/json"
	"net/http"

	"github.com/Interlocked-Labs/oracle-provider/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/sirupsen/logrus"
)

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
	r.Get("/", rs.GetGasTokenPerChain)
	return r
}

func (rs *Resource) GetGasTokenPerChain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Content-Type", "application/json")
	response, _ := rs.Oracle.CalcNSubmitTokenPrices()
	rss, _ := json.Marshal(response)
	w.Write([]byte(rss))
}
