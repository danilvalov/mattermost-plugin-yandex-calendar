package ycal

import (
	"strings"
	"testing"

	"github.com/emersion/go-ical"
)

func TestVeventToRemoteEvent_AllDay(t *testing.T) {
	t.Parallel()

	raw := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:test-uid-123@example.com
DTSTART;VALUE=DATE:20260504
DTEND;VALUE=DATE:20260505
SUMMARY:All day meeting
END:VEVENT
END:VCALENDAR`

	dec := ical.NewDecoder(strings.NewReader(raw))
	cal, err := dec.Decode()
	if err != nil {
		t.Fatal(err)
	}

	var ve *ical.Component
	for _, ch := range cal.Children {
		if ch.Name == ical.CompEvent {
			ve = ch
			break
		}
	}
	if ve == nil {
		t.Fatal("no vevent")
	}

	ev, err := veventToRemoteEvent(cal, ve)
	if err != nil {
		t.Fatal(err)
	}
	if ev.ICalUID != "test-uid-123@example.com" {
		t.Fatalf("ICalUID: %q", ev.ICalUID)
	}
	if !ev.IsAllDay {
		t.Fatal("expected all-day")
	}
	if ev.Subject != "All day meeting" {
		t.Fatalf("Subject: %q", ev.Subject)
	}
}
