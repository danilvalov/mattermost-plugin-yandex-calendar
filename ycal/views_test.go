package ycal

import (
	"strings"
	"testing"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/emersion/go-ical"
)

func TestRecurrenceInstanceKey_UsesRecurrenceID(t *testing.T) {
	t.Parallel()

	raw := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:series@example.com
DTSTART;TZID=Asia/Omsk:20260508T160000
DTEND;TZID=Asia/Omsk:20260508T170000
SUMMARY:Android Weekly
END:VEVENT
BEGIN:VEVENT
UID:series@example.com
RECURRENCE-ID;TZID=Asia/Omsk:20260508T160000
DTSTART;TZID=Asia/Omsk:20260508T160000
DTEND;TZID=Asia/Omsk:20260508T170000
SUMMARY:Android Weekly
END:VEVENT
END:VCALENDAR`

	dec := ical.NewDecoder(strings.NewReader(raw))
	cal, err := dec.Decode()
	if err != nil {
		t.Fatal(err)
	}

	if len(cal.Children) != 2 {
		t.Fatalf("expected 2 VEVENTs, got %d", len(cal.Children))
	}

	master := cal.Children[0]
	override := cal.Children[1]

	masterEvent, err := veventToRemoteEvent(cal, master)
	if err != nil {
		t.Fatal(err)
	}
	overrideEvent, err := veventToRemoteEvent(cal, override)
	if err != nil {
		t.Fatal(err)
	}

	masterKey := recurrenceInstanceKey(cal, master, masterEvent)
	overrideKey := recurrenceInstanceKey(cal, override, overrideEvent)
	if masterKey != overrideKey {
		t.Fatalf("expected same recurrence key, got master=%q override=%q", masterKey, overrideKey)
	}
}

func TestQueryRemoteEventsDedup_OverrideReplacesMaster(t *testing.T) {
	t.Parallel()

	key := "series@example.com|2026-05-08T10:00:00Z"
	master := &remote.Event{
		ICalUID: "series@example.com",
		Subject: "Android Weekly",
		Weblink: "https://calendar.yandex.ru/event?event_id=174937470324555",
	}
	override := &remote.Event{
		ICalUID: "series@example.com",
		Subject: "Android Weekly",
		Weblink: "https://calendar.yandex.ru/event?event_id=176031423254037",
	}

	out := []*remote.Event{master}
	seenByInstance := map[string]int{key: 0}
	hasOverrideByInstance := map[string]bool{key: false}

	out, seenByInstance, hasOverrideByInstance = appendWithRecurrenceDedup(
		out, seenByInstance, hasOverrideByInstance, key, true, override,
	)

	if len(out) != 1 {
		t.Fatalf("expected 1 event after dedup, got %d", len(out))
	}
	if out[0].Weblink != "https://calendar.yandex.ru/event?event_id=176031423254037" {
		t.Fatalf("expected override URL, got %q", out[0].Weblink)
	}
	if !hasOverrideByInstance[key] {
		t.Fatal("expected override marker for dedup key")
	}
	if seenByInstance[key] != 0 {
		t.Fatalf("expected index 0 for dedup key, got %d", seenByInstance[key])
	}
}
