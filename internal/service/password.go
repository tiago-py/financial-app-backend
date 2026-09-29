package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"financial-app-backend/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func validateNewPassword(current, password string) error {
	if strings.TrimSpace(password) == "" || utf8.RuneCountInString(password) < 8 || len(password) > 72 {
		return model.ValidationError{Fields: []model.FieldError{{Field: "newPassword", Message: "use no minimo 8 caracteres e no maximo 72 bytes UTF-8"}}}
	}
	if current == password {
		return model.ValidationError{Fields: []model.FieldError{{Field: "newPassword", Message: "a nova senha deve ser diferente da atual"}}}
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, ownerID, current, password string) error {
	if err := validateNewPassword(current, password); err != nil {
		return err
	}
	if len(current) == 0 || len(current) > 72 {
		return model.ValidationError{Fields: []model.FieldError{{Field: "currentPassword", Message: "senha atual incorreta"}}}
	}
	hash, _, err := s.store.PasswordState(ctx, ownerID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(current)) != nil {
		return model.ValidationError{Fields: []model.FieldError{{Field: "currentPassword", Message: "senha atual incorreta"}}}
	}
	nextHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.store.ChangePassword(ctx, ownerID, hash, string(nextHash))
}
