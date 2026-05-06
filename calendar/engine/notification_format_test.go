package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func TestLocalizeResponseStatus(t *testing.T) {
	t.Parallel()

	p := &notificationProcessor{Env: Env{Config: &config.Config{}, Dependencies: &Dependencies{}}}

	if got := p.localizeResponseStatus("u1", nil); got != "Not responded" {
		t.Fatalf("nil status: got %q", got)
	}
	if got := p.localizeResponseStatus("u1", &remote.EventResponseStatus{Response: remote.EventResponseStatusAccepted}); got != "Yes" {
		t.Fatalf("accepted: got %q", got)
	}
	if got := p.localizeResponseStatus("u1", &remote.EventResponseStatus{Response: remote.EventResponseStatusDeclined}); got != "No" {
		t.Fatalf("declined: got %q", got)
	}
	if got := p.localizeResponseStatus("u1", &remote.EventResponseStatus{Response: remote.EventResponseStatusTentative}); got != "Maybe" {
		t.Fatalf("tentative: got %q", got)
	}
}

func TestEventToFields_NilSafeAndLocalizedStatus(t *testing.T) {
	t.Parallel()

	p := &notificationProcessor{Env: Env{Config: &config.Config{}, Dependencies: &Dependencies{}}}

	ev := &remote.Event{
		Subject:        "Subject",
		BodyPreview:    "Body",
		Importance:     "",
		Start:          remote.NewDateTime(time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC), "UTC"),
		End:            remote.NewDateTime(time.Date(2026, 5, 6, 13, 0, 0, 0, time.UTC), "UTC"),
		ResponseStatus: &remote.EventResponseStatus{Response: remote.EventResponseStatusDeclined},
		Attendees: []*remote.Attendee{
			nil,
			{EmailAddress: nil},
		},
	}

	ff := p.eventToFields("u1", ev, "UTC", false)

	if got := ff[FieldResponseStatus].Strings()[0]; got != "No" {
		t.Fatalf("response status: got %q", got)
	}
	if got := ff[FieldOrganizer].Strings()[0]; got != "[Not defined](mailto:)" {
		t.Fatalf("organizer: got %q", got)
	}
	if got := ff[FieldLocation].Strings()[0]; got != "Not defined" {
		t.Fatalf("location: got %q", got)
	}
	if got := ff[FieldAttendees].Strings()[0]; got != "None" {
		t.Fatalf("attendees: got %q", got)
	}
}

func TestEventToFields_AttendeesUseNameAndMailto(t *testing.T) {
	t.Parallel()

	p := &notificationProcessor{Env: Env{Config: &config.Config{}, Dependencies: &Dependencies{}}}
	ev := &remote.Event{
		Subject: "Subject",
		Start:   remote.NewDateTime(time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC), "UTC"),
		End:     remote.NewDateTime(time.Date(2026, 5, 6, 13, 0, 0, 0, time.UTC), "UTC"),
		Attendees: []*remote.Attendee{
			{
				EmailAddress: &remote.EmailAddress{
					Name:    "Eva Toropkova",
					Address: "eva.toropkova@effective.band",
				},
			},
		},
	}

	ff := p.eventToFields("u1", ev, "UTC", false)
	if got, want := ff[FieldAttendees].Strings()[0], "[Eva Toropkova](mailto:eva.toropkova@effective.band)"; got != want {
		t.Fatalf("attendee value: got %q want %q", got, want)
	}
}

func TestEventToFields_BodyPreviewLinkifiesURLsAndEmails(t *testing.T) {
	t.Parallel()

	p := &notificationProcessor{Env: Env{Config: &config.Config{}, Dependencies: &Dependencies{}}}
	ev := &remote.Event{
		Subject: "Subject",
		Body: &remote.ItemBody{
			Content: "Join Yandex Telemost\nhttps://telemost.360.yandex.ru/j/8510081139\nContact: team@example.com",
		},
		Start: remote.NewDateTime(time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC), "UTC"),
		End:   remote.NewDateTime(time.Date(2026, 5, 6, 13, 0, 0, 0, time.UTC), "UTC"),
	}

	ff := p.eventToFields("u1", ev, "UTC", false)
	got := ff[FieldBodyPreview].Strings()[0]

	if !strings.Contains(got, "[https://telemost.360.yandex.ru/j/8510081139](https://telemost.360.yandex.ru/j/8510081139)") {
		t.Fatalf("url was not linkified: %q", got)
	}
	if !strings.Contains(got, "[team@example.com](mailto:team@example.com)") {
		t.Fatalf("email was not linkified: %q", got)
	}
}

func TestEventToFields_BodyPreviewLinkifiesTelemostMultilineRU(t *testing.T) {
	t.Parallel()

	p := &notificationProcessor{Env: Env{Config: &config.Config{}, Dependencies: &Dependencies{}}}
	ev := &remote.Event{
		Subject: "Subject",
		Body: &remote.ItemBody{
			Content: "Присоединиться Yandex Telemost\nhttps://telemost.360.yandex.ru/j/8510081139",
		},
		Start: remote.NewDateTime(time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC), "UTC"),
		End:   remote.NewDateTime(time.Date(2026, 5, 6, 13, 0, 0, 0, time.UTC), "UTC"),
	}

	ff := p.eventToFields("u1", ev, "UTC", false)
	got := ff[FieldBodyPreview].Strings()[0]

	if !strings.Contains(got, "Присоединиться Yandex Telemost\n") {
		t.Fatalf("multiline text was changed unexpectedly: %q", got)
	}
	if !strings.Contains(got, "[https://telemost.360.yandex.ru/j/8510081139](https://telemost.360.yandex.ru/j/8510081139)") {
		t.Fatalf("telemost url was not linkified: %q", got)
	}
}
