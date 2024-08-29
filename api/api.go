package api

import (
	"net/http"
	"time"

	"github.com/Interlocked-Labs/oracle-provider/logging"
	"github.com/Interlocked-Labs/oracle-provider/modules/tokentracker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
)

// New configures application resources and routes.
func New(enableCORS bool, o *tokentracker.Oracle) (*chi.Mux, error) {
	logger := logging.NewLogger()

	tokenTrackResource, err := tokentracker.NewResource(o)
	if err != nil {
		logger.WithField("module", "app").Error(err)
		return nil, err
	}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(15 * time.Second))
	r.Use(render.SetContentType(render.ContentTypeJSON))

	if enableCORS {
		r.Use(corsConfig().Handler)
	}

	r.Mount("/refresh-token-prices", tokenTrackResource.Router())

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			log.Printf("Failed to write response: %v", err)
		}
	})

	return r, nil
}

func corsConfig() *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           86400, // Maximum value not ignored by any of major browsers
	})
}
