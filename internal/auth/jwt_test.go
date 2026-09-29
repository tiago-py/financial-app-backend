package auth

import (
	"fmt"
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

func TestSessionVersion(t *testing.T) {
	manager := NewManager("uma-chave-de-testes-com-pelo-menos-32-caracteres", time.Hour)
	token, err := manager.IssueVersion("user-123", 7)
	if err != nil {
		t.Fatal(err)
	}
	userID, version, err := manager.ParseVersion(token)
	if err != nil || userID != "user-123" || version != 7 {
		t.Fatalf("ParseVersion = %q, %d, %v", userID, version, err)
	}
}

func TestLegacyTokenVersionZero(t *testing.T) {
	manager := NewManager("uma-chave-de-testes-com-pelo-menos-32-caracteres", time.Hour)
	header := encode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := encode([]byte(fmt.Sprintf(`{"sub":"legacy","exp":%d}`, time.Now().Add(time.Hour).Unix())))
	unsigned := header + "." + payload
	_, version, err := manager.ParseVersion(unsigned + "." + manager.sign(unsigned))
	if err != nil || version != 0 {
		t.Fatalf("legacy version = %d, %v", version, err)
	}
}

func TestRejectsInvalidVersions(t *testing.T) {
	manager := NewManager("uma-chave-de-testes-com-pelo-menos-32-caracteres", time.Hour)
	for _, value := range []string{`"1"`, "-1", "1.5", "null", "9007199254740992"} {
		header := encode([]byte(`{"alg":"HS256","typ":"JWT"}`))
		payload := encode([]byte(fmt.Sprintf(`{"sub":"user","exp":%d,"ver":%s}`, time.Now().Add(time.Hour).Unix(), value)))
		unsigned := header + "." + payload
		if _, _, err := manager.ParseVersion(unsigned + "." + manager.sign(unsigned)); err == nil {
			t.Fatalf("accepted invalid version %s", value)
		}
	}
}
