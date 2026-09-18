package model

import "testing"

func TestValidatePositiveCents(t *testing.T) {
	if err := ValidatePositiveCents(1, "amountCents"); err != nil {
		t.Fatalf("valor positivo foi rejeitado: %v", err)
	}
	if err := ValidatePositiveCents(0, "amountCents"); err == nil {
		t.Fatal("zero deveria ser rejeitado")
	}
	if err := ValidatePositiveCents(-1, "amountCents"); err == nil {
		t.Fatal("valor negativo deveria ser rejeitado")
	}
}

func TestValidateDate(t *testing.T) {
	if err := ValidateDate("2026-09-16", "occurredOn"); err != nil {
		t.Fatalf("data valida foi rejeitada: %v", err)
	}
	if err := ValidateDate("16/09/2026", "occurredOn"); err == nil {
		t.Fatal("data fora do formato deveria ser rejeitada")
	}
}

func TestValidateDirection(t *testing.T) {
	for _, direction := range []string{"in", "out"} {
		if err := ValidateDirection(direction); err != nil {
			t.Fatalf("direcao %q foi rejeitada: %v", direction, err)
		}
	}
	if err := ValidateDirection("transfer"); err == nil {
		t.Fatal("direcao desconhecida deveria ser rejeitada")
	}
}
