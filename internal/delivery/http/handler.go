package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/murat/quotation-service/internal/usecase"
)

type QuoteHandler struct {
	uc     usecase.QuotationUseCase
	logger *slog.Logger
}

func NewQuoteHandler(uc usecase.QuotationUseCase, logger *slog.Logger) *QuoteHandler {
	return &QuoteHandler{
		uc:     uc,
		logger: logger,
	}
}

// RequestUpdate godoc
// @Summary      Request exchange rate update
// @Description  Asks the system to fetch and update the latest exchange rate for a given pair.
// @Tags         quotes
// @Accept       json
// @Produce      json
// @Param        request body      requestUpdateRequest  true  "Pair info"
// @Success      202  {object}  requestUpdateResponse
// @Failure      400  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /quotes/update [post]
func (h *QuoteHandler) RequestUpdate(w http.ResponseWriter, r *http.Request) {
	var req requestUpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Pair == "" {
		h.writeError(w, http.StatusBadRequest, "pair is required")
		return
	}

	id, err := h.uc.RequestUpdate(r.Context(), req.Pair)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidPair) {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		h.logger.Error("failed to request update", "error", err)
		h.writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.writeJSON(w, http.StatusAccepted, requestUpdateResponse{
		RequestID: id,
	})
}

// GetStatus godoc
// @Summary      Get request status
// @Description  Returns the current status of a quote update request.
// @Tags         quotes
// @Produce      json
// @Param        id   path      string  true  "Request ID"
// @Success      200  {object}  quoteRequestResponse
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /quotes/requests/{id} [get]
func (h *QuoteHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	req, err := h.uc.GetByRequestID(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get request status", "id", id, "error", err)
		h.writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if req == nil {
		h.writeError(w, http.StatusNotFound, "request not found")
		return
	}

	h.writeJSON(w, http.StatusOK, quoteRequestResponse{
		Price:     req.Price,
		UpdatedAt: req.UpdatedAt,
	})
}

// GetLatest godoc
// @Summary      Get latest quote
// @Description  Returns the most recent exchange rate for a given pair.
// @Tags         quotes
// @Produce      json
// @Param        pair query     string  true  "Currency Pair (e.g., USD/EUR)" default(USD/KZT)
// @Success      200  {object}  latestQuoteResponse
// @Failure      400  {object}  errorResponse
// @Failure      404  {object}  errorResponse
// @Failure      500  {object}  errorResponse
// @Router       /quotes/latest [get]
func (h *QuoteHandler) GetLatest(w http.ResponseWriter, r *http.Request) {
	pair := r.URL.Query().Get("pair")
	if pair == "" {
		h.writeError(w, http.StatusBadRequest, "pair is required")
		return
	}

	quote, err := h.uc.GetLatest(r.Context(), pair)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidPair) {
			h.writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		h.logger.Error("failed to get latest quote", "pair", pair, "error", err)
		h.writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if quote == nil {
		h.writeError(w, http.StatusNotFound, "quote not found")
		return
	}

	h.writeJSON(w, http.StatusOK, latestQuoteResponse{
		Price:     quote.Price,
		UpdatedAt: quote.UpdatedAt,
	})
}
