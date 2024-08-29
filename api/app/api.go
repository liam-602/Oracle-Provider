// Package app ties together application resources and handlers.
package app

import (
	"github.com/go-chi/chi/v5"
)

// API provides application resources and handlers.
type API struct {
}

// Router provides application routes.
func (a *API) Router() *chi.Mux {
	r := chi.NewRouter()
	return r
}
