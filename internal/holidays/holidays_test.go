package holidays

import (
	"testing"
	"time"
)

func TestPlannedHoursForWeek(t *testing.T) {
	holidayWeek := time.Date(2026, 9, 9, 0, 0, 0, 0, time.Local) // quarta da semana com Sep7+Sep8
	if got := PlannedHoursForWeek(holidayWeek); got != 24 {
		t.Errorf("holiday week: got %.0f, want 24", got)
	}

	normalWeek := time.Date(2026, 9, 16, 0, 0, 0, 0, time.Local)
	if got := PlannedHoursForWeek(normalWeek); got != 40 {
		t.Errorf("normal week: got %.0f, want 40", got)
	}
}

func TestIsHoliday(t *testing.T) {
	if ok, name := IsHoliday(time.Date(2026, 9, 7, 10, 0, 0, 0, time.Local)); !ok || name != "Independência do Brasil" {
		t.Errorf("Sep 7 should be a holiday, got ok=%v name=%q", ok, name)
	}
	if ok, _ := IsHoliday(time.Date(2026, 9, 9, 10, 0, 0, 0, time.Local)); ok {
		t.Errorf("Sep 9 should not be a holiday")
	}
}
