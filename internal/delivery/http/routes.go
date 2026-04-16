package http

import (
	"net/http"

	_ "github.com/murat/quotation-service/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

func (h *QuoteHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /quotes/update", h.RequestUpdate)
	mux.HandleFunc("GET /quotes/requests/{id}", h.GetStatus)
	mux.HandleFunc("GET /quotes/latest", h.GetLatest)

	mux.Handle("GET /docs/", httpSwagger.Handler(
		httpSwagger.URL("/docs/doc.json"),
	))
}
