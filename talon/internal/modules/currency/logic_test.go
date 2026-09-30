package currency

import "testing"

func TestRound(t *testing.T) {
	if got := Round(1.234, 0.01); got != 1.23 {
		t.Errorf("Round(1.234, 0.01) = %v, want 1.23", got)
	}
	if got := Round(2.5, 1); got != 2 {
		t.Errorf("Round(2.5, 1) = %v, want 2 (half-even)", got)
	}
	if got := Round(3.5, 1); got != 4 {
		t.Errorf("Round(3.5, 1) = %v, want 4 (half-even)", got)
	}
	if got := Round(5.0, 0); got != 5.0 {
		t.Errorf("Round(x, 0) = %v, want unchanged", got)
	}
	if got := Round(1.005, 0.05); got != 1.0 {
		t.Errorf("Round(1.005, 0.05) = %v, want 1.0", got)
	}
}

func TestIsZero(t *testing.T) {
	if !IsZero(0.001, 0.01) {
		t.Error("IsZero(0.001, 0.01) = false, want true")
	}
	if IsZero(0.01, 0.01) {
		t.Error("IsZero(0.01, 0.01) = true, want false")
	}
	if !IsZero(0, 0) {
		t.Error("IsZero(0, 0) = false, want true")
	}
}
