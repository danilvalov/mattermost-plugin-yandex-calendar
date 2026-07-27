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

func TestPatchVEVENTFromRemote_RewriteAttendees(t *testing.T) {
	t.Parallel()

	cal, ve := mustDecodeVEVENT(t, `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:patch-att@example.com
DTSTART:20260504T100000Z
DTEND:20260504T110000Z
SUMMARY:Meet
ORGANIZER:mailto:org@example.com
ATTENDEE;PARTSTAT=ACCEPTED;CN=Keep Me;RSVP=FALSE;ROLE=CHAIR;CUTYPE=INDIVIDUAL:mailto:keep@example.com
ATTENDEE;PARTSTAT=NEEDS-ACTION:mailto:drop@example.com
END:VEVENT
END:VCALENDAR`)

	loc := time.UTC
	in := &remote.Event{
		Subject:          "Meet",
		IsAllDay:         false,
		Start:            remote.NewDateTime(time.Date(2026, 5, 4, 10, 0, 0, 0, loc), "UTC"),
		End:              remote.NewDateTime(time.Date(2026, 5, 4, 11, 0, 0, 0, loc), "UTC"),
		RewriteAttendees: true,
		Attendees: []*remote.Attendee{
			{EmailAddress: &remote.EmailAddress{Address: "keep@example.com"}},
			{EmailAddress: &remote.EmailAddress{Address: "new@example.com", Name: "New"}},
		},
	}
	if err := patchVEVENTFromRemote(cal, ve, in); err != nil {
		t.Fatal(err)
	}
	atts := ve.Props.Values(ical.PropAttendee)
	if len(atts) != 2 {
		t.Fatalf("attendees: %d", len(atts))
	}
	byMail := map[string]*ical.Prop{}
	for i := range atts {
		byMail[strings.ToLower(atts[i].Value)] = &atts[i]
	}
	keep := byMail["mailto:keep@example.com"]
	if keep == nil || keep.Params.Get(ical.ParamParticipationStatus) != "ACCEPTED" {
		t.Fatalf("keep PARTSTAT: %#v", keep)
	}
	if keep.Params.Get(ical.ParamRSVP) != "FALSE" {
		t.Fatalf("keep RSVP: %#v", keep)
	}
	if keep.Params.Get(ical.ParamRole) != "CHAIR" || keep.Params.Get(ical.ParamCalendarUserType) != "INDIVIDUAL" {
		t.Fatalf("keep ROLE/CUTYPE: %#v", keep)
	}
	nw := byMail["mailto:new@example.com"]
	if nw == nil || nw.Params.Get(ical.ParamParticipationStatus) != "NEEDS-ACTION" {
		t.Fatalf("new PARTSTAT: %#v", nw)
	}
	if nw.Params.Get(ical.ParamRSVP) != "TRUE" {
		t.Fatalf("new RSVP: %#v", nw)
	}
	if _, ok := byMail["mailto:drop@example.com"]; ok {
		t.Fatal("drop should be gone")
	}
}

func TestAssertEventUpdateApplied_Attendees(t *testing.T) {
	t.Parallel()
	err := assertEventUpdateApplied(
		&remote.Event{
			RewriteAttendees: true,
			Attendees: []*remote.Attendee{
				{EmailAddress: &remote.EmailAddress{Address: "a@example.com"}},
			},
		},
		&remote.Event{
			Attendees: []*remote.Attendee{
				{EmailAddress: &remote.EmailAddress{Address: "b@example.com"}},
			},
		},
	)
	if err == nil {
		t.Fatal("expected attendees mismatch error")
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

func TestBuildCalendarFromRemoteEvent_RequireTelemost(t *testing.T) {
	t.Parallel()
	in := &remote.Event{
		Subject:         "Meet",
		RequireTelemost: true,
		Start:           remote.NewDateTime(time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC), "UTC"),
		End:             remote.NewDateTime(time.Date(2026, 7, 28, 11, 0, 0, 0, time.UTC), "UTC"),
	}
	cal, err := buildCalendarFromRemoteEvent(in, "uid-telemost", "a@example.com")
	if err != nil {
		t.Fatal(err)
	}
	ve := firstVEVENT(cal)
	if got := propText(ve, "X-TELEMOST-REQUIRED"); got != "TRUE" {
		t.Fatalf("X-TELEMOST-REQUIRED: %q", got)
	}
}

func TestPatchVEVENTFromRemote_RequireTelemost(t *testing.T) {
	t.Parallel()

	cal, ve := mustDecodeVEVENT(t, `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:patch-telemost@example.com
DTSTART:20260728T100000Z
DTEND:20260728T110000Z
SUMMARY:No conf
X-TELEMOST-REQUIRED:TRUE
END:VEVENT
END:VCALENDAR`)

	in := &remote.Event{
		Subject:         "No conf",
		RequireTelemost: true,
		Start:           remote.NewDateTime(time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC), "UTC"),
		End:             remote.NewDateTime(time.Date(2026, 7, 28, 11, 0, 0, 0, time.UTC), "UTC"),
	}
	if err := patchVEVENTFromRemote(cal, ve, in); err != nil {
		t.Fatal(err)
	}
	if got := propText(ve, "X-TELEMOST-REQUIRED"); got != "TRUE" {
		t.Fatalf("X-TELEMOST-REQUIRED: %q", got)
	}

	// Already has conference — clear stale mint hint, do not re-request.
	cal2, ve2 := mustDecodeVEVENT(t, `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:patch-telemost-existing@example.com
DTSTART:20260728T100000Z
DTEND:20260728T110000Z
SUMMARY:Has conf
X-TELEMOST-REQUIRED:TRUE
X-TELEMOST-CONFERENCE:https://telemost.yandex.ru/j/1
END:VEVENT
END:VCALENDAR`)
	in2 := &remote.Event{
		Subject:         "Has conf",
		RequireTelemost: true,
		Start:           remote.NewDateTime(time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC), "UTC"),
		End:             remote.NewDateTime(time.Date(2026, 7, 28, 11, 0, 0, 0, time.UTC), "UTC"),
	}
	if err := patchVEVENTFromRemote(cal2, ve2, in2); err != nil {
		t.Fatal(err)
	}
	if got := propText(ve2, "X-TELEMOST-REQUIRED"); got != "" {
		t.Fatalf("expected no X-TELEMOST-REQUIRED when conference exists, got %q", got)
	}

	// Ordinary edit clears a leftover mint hint.
	cal3, ve3 := mustDecodeVEVENT(t, `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:patch-telemost-clear@example.com
DTSTART:20260728T100000Z
DTEND:20260728T110000Z
SUMMARY:Clear
X-TELEMOST-REQUIRED:TRUE
END:VEVENT
END:VCALENDAR`)
	in3 := &remote.Event{
		Subject: "Clear",
		Start:   remote.NewDateTime(time.Date(2026, 7, 28, 10, 0, 0, 0, time.UTC), "UTC"),
		End:     remote.NewDateTime(time.Date(2026, 7, 28, 11, 0, 0, 0, time.UTC), "UTC"),
	}
	if err := patchVEVENTFromRemote(cal3, ve3, in3); err != nil {
		t.Fatal(err)
	}
	if got := propText(ve3, "X-TELEMOST-REQUIRED"); got != "" {
		t.Fatalf("expected leftover X-TELEMOST-REQUIRED cleared, got %q", got)
	}
}

func TestAssertTelemostApplied(t *testing.T) {
	t.Parallel()
	if err := assertTelemostApplied(&remote.Event{RequireTelemost: true}, &remote.Event{}); err == nil {
		t.Fatal("expected missing conference error")
	}
	if err := assertTelemostApplied(
		&remote.Event{RequireTelemost: true},
		&remote.Event{Conference: &remote.Conference{URL: "https://telemost.yandex.ru/j/1"}},
	); err != nil {
		t.Fatal(err)
	}
	if err := assertTelemostApplied(&remote.Event{}, &remote.Event{}); err != nil {
		t.Fatal(err)
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
