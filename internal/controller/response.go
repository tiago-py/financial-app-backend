package controller

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"financial-app-backend/internal/model"
)

type errorBody struct {
	Code        string             `json:"code"`
	Message     string             `json:"message"`
	FieldErrors []model.FieldError `json:"fieldErrors,omitempty"`
	RequestID   string             `json:"requestId,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if status != http.StatusNoContent {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func decodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	body := errorBody{Code: "internal_error", Message: "Nao foi possivel concluir a operacao.", RequestID: requestID(r.Context())}
	var validation model.ValidationError
	switch {
	case errors.As(err, &validation):
		status = http.StatusUnprocessableEntity
		body.Code, body.Message, body.FieldErrors = "validation_error", "Revise os campos informados.", validation.Fields
	case errors.Is(err, model.ErrUnauthorized):
		status = http.StatusUnauthorized
		body.Code, body.Message = "unauthorized", "Credenciais invalidas ou sessao expirada."
	case errors.Is(err, model.ErrForbidden):
		status = http.StatusForbidden
		body.Code, body.Message = "forbidden", "Voce nao pode acessar este recurso."
	case errors.Is(err, model.ErrNotFound):
		status = http.StatusNotFound
		body.Code, body.Message = "not_found", "Recurso nao encontrado."
	case errors.Is(err, model.ErrIdempotencyKey):
		status = http.StatusConflict
		body.Code, body.Message = "idempotency_conflict", "A chave de idempotencia ja foi usada com outros dados."
	case errors.Is(err, model.ErrConflict):
		status = http.StatusConflict
		body.Code, body.Message = "conflict", "A operacao conflita com o estado atual do recurso."
	case errors.Is(err, model.ErrInvalid):
		status = http.StatusUnprocessableEntity
		body.Code, body.Message = "business_rule", "A operacao viola uma regra financeira."
	}
	writeJSON(w, status, body)
}

func badJSON(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusBadRequest, errorBody{
		Code: "invalid_json", Message: "O corpo JSON e invalido.", RequestID: requestID(r.Context()),
	})
}
