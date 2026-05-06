package ycal

import (
	"strings"
	"testing"
	"time"

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

func TestVeventToRemoteEvent_PreferTZIDOverHistoricalVTIMEZONEOffset(t *testing.T) {
	t.Parallel()

	raw := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VTIMEZONE
TZID:Europe/Moscow
BEGIN:STANDARD
TZOFFSETFROM:+0300
TZOFFSETTO:+0230
DTSTART:19190701T000000
END:STANDARD
END:VTIMEZONE
BEGIN:VEVENT
UID:test-uid-moscow@example.com
DTSTART;TZID=Europe/Moscow:20260504T140000
DTEND;TZID=Europe/Moscow:20260504T143000
SUMMARY:Meeting
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

	if ev.Start == nil || ev.End == nil {
		t.Fatal("missing event datetime")
	}

	start := ev.Start.Time()
	end := ev.End.Time()

	if got, want := start.Hour(), 14; got != want {
		t.Fatalf("start hour: got %d want %d", got, want)
	}
	if got, want := start.Minute(), 0; got != want {
		t.Fatalf("start minute: got %d want %d", got, want)
	}
	if got, want := end.Hour(), 14; got != want {
		t.Fatalf("end hour: got %d want %d", got, want)
	}
	if got, want := end.Minute(), 30; got != want {
		t.Fatalf("end minute: got %d want %d", got, want)
	}

	if d := end.Sub(start); d != 30*time.Minute {
		t.Fatalf("duration: got %s want 30m", d)
	}
}

func TestVeventToRemoteEvent_ZuluDateTimeIgnoresTZID(t *testing.T) {
	t.Parallel()

	raw := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:test-uid-zulu@example.com
DTSTART;TZID=Asia/Omsk:20260504T140000Z
DTEND;TZID=Asia/Omsk:20260504T143000Z
SUMMARY:Meeting
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
	if ev.Start == nil || ev.End == nil {
		t.Fatal("missing event datetime")
	}

	if got, want := ev.Start.TimeZone, "UTC"; got != want {
		t.Fatalf("start timezone: got %q want %q", got, want)
	}
	if got, want := ev.End.TimeZone, "UTC"; got != want {
		t.Fatalf("end timezone: got %q want %q", got, want)
	}

	start := ev.Start.Time()
	end := ev.End.Time()

	if got, want := start.Hour(), 14; got != want {
		t.Fatalf("start hour: got %d want %d", got, want)
	}
	if got, want := end.Hour(), 14; got != want {
		t.Fatalf("end hour: got %d want %d", got, want)
	}
	if d := end.Sub(start); d != 30*time.Minute {
		t.Fatalf("duration: got %s want 30m", d)
	}
}

func TestVeventToRemoteEvent_MapsURLAndDescription(t *testing.T) {
	t.Parallel()

	raw := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:test-uid-url@example.com
DTSTART:20260504T120000Z
DTEND:20260504T130000Z
SUMMARY:Meeting
DESCRIPTION:Присоединиться Yandex Telemost\nhttps://telemost.360.yandex.ru/j/8510081139
URL:https://calendar.yandex.ru/event?event_id=177625219630211
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

	if got, want := ev.Weblink, "https://calendar.yandex.ru/event?event_id=177625219630211"; got != want {
		t.Fatalf("weblink: got %q want %q", got, want)
	}
	if ev.Body == nil {
		t.Fatal("missing body")
	}
	if got, want := ev.Body.Content, "Присоединиться Yandex Telemost\nhttps://telemost.360.yandex.ru/j/8510081139"; got != want {
		t.Fatalf("body content: got %q want %q", got, want)
	}
	if got, want := ev.BodyPreview, "Присоединиться Yandex Telemost\nhttps://telemost.360.yandex.ru/j/8510081139"; got != want {
		t.Fatalf("body preview: got %q want %q", got, want)
	}
}
