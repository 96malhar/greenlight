package main

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/v1/healthcheck", app.healthcheckHandler)
	r.Post("/v1/movies", app.createMovieHandler)
	r.Get("/v1/movies/{id}", app.showMovieHandler)

	return r
}
