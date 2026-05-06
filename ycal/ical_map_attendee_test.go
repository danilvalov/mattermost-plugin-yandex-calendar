package ycal

import (
	"strings"
	"testing"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/emersion/go-ical"
)

func TestAttendeeDisplayName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		mail, cn, want string
	}{
		{"alice@example.com", "", "alice"},
		{"alice@example.com", "Alice Q.", "Alice Q."},
		{"", "", ""},
		{"nohost", "", "nohost"},
	}
	for _, tc := range tests {
		if g := attendeeDisplayName(tc.mail, tc.cn); g != tc.want {
			t.Fatalf("attendeeDisplayName(%q, %q) = %q, want %q", tc.mail, tc.cn, g, tc.want)
		}
	}
}

func TestVeventToRemoteEvent_AttendeeNames(t *testing.T) {
	t.Parallel()
	raw := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:ev@example.com
DTSTART:20260504T120000Z
DTEND:20260504T130000Z
SUMMARY:Meet
ORGANIZER:mailto:boss@example.com
ATTENDEE;PARTSTAT=ACCEPTED:mailto:plain@example.com
ATTENDEE;CN=Named User;PARTSTAT=TENTATIVE:mailto:named@example.com
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
	if ev.Organizer == nil || ev.Organizer.EmailAddress == nil {
		t.Fatal("missing organizer")
	}
	if g, w := ev.Organizer.EmailAddress.Name, "boss"; g != w {
		t.Fatalf("organizer Name: got %q want %q", g, w)
	}
	if len(ev.Attendees) != 2 {
		t.Fatalf("attendees len: %d", len(ev.Attendees))
	}
	if g, w := ev.Attendees[0].EmailAddress.Name, "plain"; g != w {
		t.Fatalf("attendee0 Name: got %q want %q", g, w)
	}
	if g, w := ev.Attendees[1].EmailAddress.Name, "Named User"; g != w {
		t.Fatalf("attendee1 Name: got %q want %q", g, w)
	}
}

func TestApplyCurrentUserContext(t *testing.T) {
	t.Parallel()

	ev := &remote.Event{
		Organizer: &remote.Attendee{
			EmailAddress: &remote.EmailAddress{Address: "boss@example.com"},
		},
		ResponseStatus: &remote.EventResponseStatus{Response: remote.EventResponseStatusNotAnswered},
		Attendees: []*remote.Attendee{
			{
				EmailAddress: &remote.EmailAddress{Address: "member@example.com"},
				Status:       &remote.EventResponseStatus{Response: remote.EventResponseStatusAccepted},
			},
		},
	}

	applyCurrentUserContext(ev, "member@example.com")

	if ev.IsOrganizer {
		t.Fatal("expected not organizer")
	}
	if ev.ResponseStatus == nil || ev.ResponseStatus.Response != remote.EventResponseStatusAccepted {
		t.Fatalf("unexpected response status: %#v", ev.ResponseStatus)
	}
}
