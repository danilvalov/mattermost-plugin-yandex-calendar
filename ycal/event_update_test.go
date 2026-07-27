package ycal

import (
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-ical"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func TestPatchVEVENTFromRemote_Timed(t *testing.T) {
	t.Parallel()

	cal, ve := mustDecodeVEVENT(t, `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:patch-timed@example.com
DTSTART:20260504T100000Z
DTEND:20260504T110000Z
SUMMARY:Old
DESCRIPTION:Keep me
ORGANIZER:mailto:a@example.com
ATTENDEE;PARTSTAT=ACCEPTED:mailto:b@example.com
END:VEVENT
END:VCALENDAR`)

	in := &remote.Event{
		Subject:  "New title",
		IsAllDay: false,
		Start:    remote.NewDateTime(time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC), "UTC"),
		End:      remote.NewDateTime(time.Date(2026, 5, 4, 13, 30, 0, 0, time.UTC), "UTC"),
		Body:     &remote.ItemBody{Content: "Updated desc"},
		Location: &remote.Location{DisplayName: "Room"},
	}
	if err := patchVEVENTFromRemote(cal, ve, in); err != nil {
		t.Fatal(err)
	}

	if sum, _ := ve.Props.Text(ical.PropSummary); sum != "New title" {
		t.Fatalf("summary: %q", sum)
	}
	if desc, _ := ve.Props.Text(ical.PropDescription); desc != "Updated desc" {
		t.Fatalf("description: %q", desc)
	}
	if loc, _ := ve.Props.Text(ical.PropLocation); loc != "Room" {
		t.Fatalf("location: %q", loc)
	}
	if ve.Props.Get(ical.PropOrganizer) == nil {
		t.Fatal("expected ORGANIZER preserved")
	}
	if len(ve.Props.Values(ical.PropAttendee)) != 1 {
		t.Fatal("expected ATTENDEE preserved")
	}
	ds := ve.Props.Get(ical.PropDateTimeStart)
	if ds == nil || !strings.Contains(ds.Value, "120000") {
		t.Fatalf("DTSTART: %#v", ds)
	}
	if seq, _ := ve.Props.Text(ical.PropSequence); seq != "1" {
		t.Fatalf("SEQUENCE: %q", seq)
	}
}

func TestPatchVEVENTFromRemote_PreservesTimesBumpsSequence(t *testing.T) {
	t.Parallel()

	cal, ve := mustDecodeVEVENT(t, `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:patch-preserve@example.com
DTSTART;TZID=Europe/Moscow:20260728T223000
DTEND;TZID=Europe/Moscow:20260728T230000
SUMMARY:Old
SEQUENCE:0
END:VEVENT
END:VCALENDAR`)

	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	in := &remote.Event{
		Subject:  "New title",
		IsAllDay: false,
		Start:    remote.NewDateTime(time.Date(2026, 7, 28, 22, 30, 0, 0, loc), "Europe/Moscow"),
		End:      remote.NewDateTime(time.Date(2026, 7, 28, 23, 0, 0, 0, loc), "Europe/Moscow"),
	}
	if err := patchVEVENTFromRemote(cal, ve, in); err != nil {
		t.Fatal(err)
	}
	ds := ve.Props.Get(ical.PropDateTimeStart)
	if ds == nil || ds.Params.Get(ical.ParamTimezoneID) != "Europe/Moscow" || ds.Value != "20260728T223000" {
		t.Fatalf("DTSTART should be preserved, got %#v", ds)
	}
	if sum, _ := ve.Props.Text(ical.PropSummary); sum != "New title" {
		t.Fatalf("summary: %q", sum)
	}
	if seq, _ := ve.Props.Text(ical.PropSequence); seq != "1" {
		t.Fatalf("SEQUENCE: %q", seq)
	}
}

func TestAssertEventUpdateApplied(t *testing.T) {
	t.Parallel()
	err := assertEventUpdateApplied(
		&remote.Event{Subject: "New"},
		&remote.Event{Subject: "Old"},
	)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPatchVEVENTFromRemote_AllDayExclusive(t *testing.T) {
	t.Parallel()

	cal, ve := mustDecodeVEVENT(t, `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:patch-allday@example.com
DTSTART;VALUE=DATE:20260504
DTEND;VALUE=DATE:20260505
SUMMARY:Day
END:VEVENT
END:VCALENDAR`)

	// Inclusive UI range May 4–May 5 → exclusive end May 6 when passed as end date May 6 midnight,
	// or same-day 23:59 should normalize to +1 day exclusive.
	in := &remote.Event{
		Subject:  "Day",
		IsAllDay: true,
		Start:    remote.NewDateTime(time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC), "UTC"),
		End:      remote.NewDateTime(time.Date(2026, 5, 4, 23, 59, 59, 0, time.UTC), "UTC"),
	}
	if err := patchVEVENTFromRemote(cal, ve, in); err != nil {
		t.Fatal(err)
	}
	de := ve.Props.Get(ical.PropDateTimeEnd)
	if de == nil || de.Value != "20260505" {
		t.Fatalf("expected exclusive DTEND 20260505, got %#v", de)
	}
}

func TestVeventToRemoteEvent_IsRecurringRRULE(t *testing.T) {
	t.Parallel()
	cal, ve := mustDecodeVEVENT(t, `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:recur@example.com
DTSTART:20260504T100000Z
DTEND:20260504T110000Z
RRULE:FREQ=WEEKLY
SUMMARY:Weekly
END:VEVENT
END:VCALENDAR`)
	ev, err := veventToRemoteEvent(cal, ve)
	if err != nil {
		t.Fatal(err)
	}
	if !ev.IsRecurring {
		t.Fatal("expected IsRecurring for RRULE")
	}
}

func TestVeventToRemoteEvent_IsRecurringRecurrenceID(t *testing.T) {
	t.Parallel()
	cal, ve := mustDecodeVEVENT(t, `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:recur@example.com
RECURRENCE-ID:20260511T100000Z
DTSTART:20260511T100000Z
DTEND:20260511T110000Z
SUMMARY:Override
END:VEVENT
END:VCALENDAR`)
	ev, err := veventToRemoteEvent(cal, ve)
	if err != nil {
		t.Fatal(err)
	}
	if !ev.IsRecurring {
		t.Fatal("expected IsRecurring for RECURRENCE-ID")
	}
}

func TestBuildCalendarFromRemoteEvent_AllDayExclusive(t *testing.T) {
	t.Parallel()
	in := &remote.Event{
		Subject:  "All day",
		IsAllDay: true,
		Start:    remote.NewDateTime(time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC), "UTC"),
		End:      remote.NewDateTime(time.Date(2026, 5, 4, 23, 59, 59, 99, time.UTC), "UTC"),
	}
	cal, err := buildCalendarFromRemoteEvent(in, "uid-1", "a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	ve := firstVEVENT(cal)
	de := ve.Props.Get(ical.PropDateTimeEnd)
	if de == nil || de.Value != "20260505" {
		t.Fatalf("expected DTEND 20260505, got %#v", de)
	}
}

func mustDecodeVEVENT(t *testing.T, raw string) (*ical.Calendar, *ical.Component) {
	t.Helper()
	cal, err := ical.NewDecoder(strings.NewReader(raw)).Decode()
	if err != nil {
		t.Fatal(err)
	}
	ve := firstVEVENT(cal)
	if ve == nil {
		t.Fatal("no VEVENT")
	}
	return cal, ve
}
