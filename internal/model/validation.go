package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound       = errors.New("recurso nao encontrado")
	ErrConflict       = errors.New("conflito de dados")
	ErrUnauthorized   = errors.New("nao autorizado")
	ErrForbidden      = errors.New("acesso negado")
	ErrInvalid        = errors.New("dados invalidos")
	ErrIdempotencyKey = errors.New("chave de idempotencia invalida")
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationError struct {
	Fields []FieldError
}

func (e ValidationError) Error() string { return "dados invalidos" }

func ValidateRequired(value, field string, max int) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return ValidationError{Fields: []FieldError{{Field: field, Message: "campo obrigatorio"}}}
	}
	if len(value) > max {
		return ValidationError{Fields: []FieldError{{Field: field, Message: fmt.Sprintf("maximo de %d caracteres", max)}}}
	}
	return nil
}

func ValidatePositiveCents(value int64, field string) error {
	if value <= 0 {
		return ValidationError{Fields: []FieldError{{Field: field, Message: "deve ser maior que zero"}}}
	}
	return nil
}

func ValidateDate(value, field string) error {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return ValidationError{Fields: []FieldError{{Field: field, Message: "use o formato YYYY-MM-DD"}}}
	}
	return nil
}

func ValidateDirection(value string) error {
	if value != "in" && value != "out" {
		return ValidationError{Fields: []FieldError{{Field: "direction", Message: "use in ou out"}}}
	}
	return nil
}
