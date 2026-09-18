package auth

import (
	"testing"
	"time"
)

func TestIssueAndParse(t *testing.T) {
	manager := NewManager("uma-chave-de-testes-com-pelo-menos-32-caracteres", time.Hour)
	token, err := manager.Issue("user-123")
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	userID, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if userID != "user-123" {
		t.Fatalf("Parse() userID = %q, want %q", userID, "user-123")
	}
}

func TestRejectsChangedToken(t *testing.T) {
	manager := NewManager("uma-chave-de-testes-com-pelo-menos-32-caracteres", time.Hour)
	token, _ := manager.Issue("user-123")
	if _, err := manager.Parse(token + "x"); err == nil {
		t.Fatal("Parse() deveria rejeitar token alterado")
	}
}

func TestRejectsExpiredToken(t *testing.T) {
	manager := NewManager("uma-chave-de-testes-com-pelo-menos-32-caracteres", -time.Second)
	token, _ := manager.Issue("user-123")
	if _, err := manager.Parse(token); err == nil {
		t.Fatal("Parse() deveria rejeitar token expirado")
	}
}
