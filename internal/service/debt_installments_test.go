package service

import "testing"

func TestGenerateDebtInstallmentsPreservesPrincipalAndClampsMonthEnd(t *testing.T) {
	firstDueDate := "2026-01-31"
	items, err := GenerateDebtInstallments(10000, 3, &firstDueDate)
	if err != nil {
		t.Fatalf("GenerateDebtInstallments returned error: %v", err)
	}
	wantAmounts := []int64{3334, 3333, 3333}
	wantDates := []string{"2026-01-31", "2026-02-28", "2026-03-31"}
	var total int64
	for index, item := range items {
		total += item.AmountCents
		if item.Number != index+1 || item.AmountCents != wantAmounts[index] || item.DueDate != wantDates[index] {
			t.Errorf("installment %d = %+v, want amount %d and date %s", index, item, wantAmounts[index], wantDates[index])
		}
	}
	if total != 10000 {
		t.Errorf("total = %d, want 10000", total)
	}
}

func TestGenerateDebtInstallmentsRequiresDueDateForMultipleInstallments(t *testing.T) {
	if _, err := GenerateDebtInstallments(10000, 2, nil); err == nil {
		t.Fatal("expected an error for missing first due date")
	}
}

func TestGenerateDebtInstallmentsLeavesSingleDebtWithoutSchedule(t *testing.T) {
	items, err := GenerateDebtInstallments(10000, 1, nil)
	if err != nil || len(items) != 0 {
		t.Fatalf("items = %+v, err = %v; want empty schedule", items, err)
	}
}
