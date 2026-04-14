package cron

import (
	"testing"
)

func TestParse_Shortcuts(t *testing.T) {
	tests := []struct {
		expr      string
		runAtLoad bool
		numSlots  int
	}{
		{"@reboot", true, 0},
		{"@hourly", false, 1},   // minute=0, hour=*
		{"@daily", false, 1},    // minute=0, hour=0
		{"@midnight", false, 1}, // alias of @daily
		{"@weekly", false, 1},   // minute=0, hour=0, weekday=0
		{"@monthly", false, 1},  // minute=0, hour=0, day=1
		{"@yearly", false, 1},
		{"@annually", false, 1},
	}

	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			s, err := Parse(tc.expr)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tc.expr, err)
			}
			if s.RunAtLoad != tc.runAtLoad {
				t.Errorf("RunAtLoad = %v, want %v", s.RunAtLoad, tc.runAtLoad)
			}
			if len(s.Intervals) != tc.numSlots {
				t.Errorf("len(Intervals) = %d, want %d", len(s.Intervals), tc.numSlots)
			}
		})
	}
}

func TestParse_SimpleSingleValues(t *testing.T) {
	s, err := Parse("0 9 * * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Intervals) != 1 {
		t.Fatalf("len(Intervals) = %d, want 1", len(s.Intervals))
	}
	iv := s.Intervals[0]
	if iv.Minute == nil || *iv.Minute != 0 {
		t.Errorf("Minute = %v, want 0", iv.Minute)
	}
	if iv.Hour == nil || *iv.Hour != 9 {
		t.Errorf("Hour = %v, want 9", iv.Hour)
	}
	if iv.Day != nil || iv.Month != nil || iv.Weekday != nil {
		t.Errorf("wildcard fields should be nil, got %+v", iv)
	}
}

func TestParse_Range(t *testing.T) {
	// Weekdays 1-5 (Mon-Fri) at 09:00.
	s, err := Parse("0 9 * * 1-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Intervals) != 5 {
		t.Fatalf("len(Intervals) = %d, want 5", len(s.Intervals))
	}
	// Check weekday values are 1..5
	got := make(map[int]bool)
	for _, iv := range s.Intervals {
		if iv.Weekday == nil {
			t.Fatalf("Weekday should not be nil")
		}
		got[*iv.Weekday] = true
	}
	for i := 1; i <= 5; i++ {
		if !got[i] {
			t.Errorf("missing weekday %d", i)
		}
	}
}

func TestParse_List(t *testing.T) {
	// 09:00 and 17:00 daily.
	s, err := Parse("0 9,17 * * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Intervals) != 2 {
		t.Fatalf("len(Intervals) = %d, want 2", len(s.Intervals))
	}
	seen := map[int]bool{}
	for _, iv := range s.Intervals {
		if iv.Hour == nil {
			t.Fatalf("Hour should not be nil")
		}
		seen[*iv.Hour] = true
	}
	if !seen[9] || !seen[17] {
		t.Errorf("expected hours 9 and 17, got %v", seen)
	}
}

func TestParse_Step(t *testing.T) {
	// Every 15 minutes.
	s, err := Parse("*/15 * * * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Intervals) != 4 {
		t.Fatalf("len(Intervals) = %d, want 4 (0,15,30,45)", len(s.Intervals))
	}
	want := map[int]bool{0: false, 15: false, 30: false, 45: false}
	for _, iv := range s.Intervals {
		if iv.Minute == nil {
			t.Fatalf("Minute should not be nil")
		}
		want[*iv.Minute] = true
	}
	for k, v := range want {
		if !v {
			t.Errorf("missing minute %d", k)
		}
	}
}

func TestParse_RangeWithStep(t *testing.T) {
	// 9,11,13,15,17 — business hours every 2h.
	s, err := Parse("0 9-17/2 * * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.Intervals) != 5 {
		t.Fatalf("len(Intervals) = %d, want 5", len(s.Intervals))
	}
}

func TestParse_WeekdaySevenIsSunday(t *testing.T) {
	s, err := Parse("0 0 * * 0,7")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 0 and 7 both mean Sunday — should dedupe to one interval.
	if len(s.Intervals) != 1 {
		t.Fatalf("len(Intervals) = %d, want 1 (0 and 7 both = Sunday)", len(s.Intervals))
	}
	if *s.Intervals[0].Weekday != 0 {
		t.Errorf("Weekday = %d, want 0", *s.Intervals[0].Weekday)
	}
}

func TestParse_Errors(t *testing.T) {
	bad := []string{
		"",
		"not a cron",
		"0 0 0 0",           // too few fields
		"0 0 * * * *",       // too many fields
		"60 0 * * *",        // minute out of range
		"0 24 * * *",        // hour out of range
		"0 0 32 * *",        // day out of range
		"0 0 * 13 *",        // month out of range
		"0 0 * * 8",         // weekday out of range
		"0 0 * * 1-5/0",     // zero step
		"@unknown",          // unknown shortcut
		"abc 0 * * *",       // non-numeric
		"5-3 0 * * *",       // inverted range
	}
	for _, expr := range bad {
		t.Run(expr, func(t *testing.T) {
			if _, err := Parse(expr); err == nil {
				t.Errorf("Parse(%q) expected error, got nil", expr)
			}
		})
	}
}

func TestParse_PulseRealFixtures(t *testing.T) {
	// The four real cron jobs from Home/schedule/.
	fixtures := []struct {
		name      string
		expr      string
		wantSlots int
	}{
		{"cron-news", "57 9 * * *", 1},
		{"cron-briefing", "3 10 * * 1-5", 5},
		{"cron-activity", "5 10 * * *", 1},
		{"cron-optimize", "1 12 * * 5", 1},
	}
	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			s, err := Parse(f.expr)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", f.expr, err)
			}
			if len(s.Intervals) != f.wantSlots {
				t.Errorf("len(Intervals) = %d, want %d", len(s.Intervals), f.wantSlots)
			}
		})
	}
}
