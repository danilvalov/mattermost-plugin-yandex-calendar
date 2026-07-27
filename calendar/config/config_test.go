package config

import "testing"

func TestCalendarUIEnabled_DefaultTrue(t *testing.T) {
	t.Parallel()
	var c StoredConfig
	if !c.CalendarUIEnabled() {
		t.Fatal("nil EnableCalendarUI must default to enabled")
	}
}

func TestCalendarUIEnabled_Explicit(t *testing.T) {
	t.Parallel()
	off := false
	on := true
	if (StoredConfig{EnableCalendarUI: &off}).CalendarUIEnabled() {
		t.Fatal("want disabled")
	}
	if !(StoredConfig{EnableCalendarUI: &on}).CalendarUIEnabled() {
		t.Fatal("want enabled")
	}
}
