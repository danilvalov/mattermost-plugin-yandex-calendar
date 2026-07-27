package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func TestEventResponsePostActions(t *testing.T) {
	t.Parallel()

	env := Env{Config: &config.Config{PluginURL: "http://localhost:8065/plugins/x"}, Dependencies: &Dependencies{}}
	actions := EventResponsePostActions(env, "u1", "ev1", remote.EventResponseStatusAccepted, "http://localhost:8065/plugins/x/action/respond")
	if len(actions) != 3 {
		t.Fatalf("want 3 actions, got %d", len(actions))
	}
	if actions[0].Type != model.PostActionTypeButton {
		t.Fatalf("want button type, got %q", actions[0].Type)
	}
	if actions[0].Style != "primary" || !actions[0].Disabled || actions[0].Integration == nil {
		t.Fatalf("selected Going should be primary+disabled with integration: %#v", actions[0])
	}
	if actions[1].Disabled || actions[1].Integration == nil || actions[2].Integration == nil {
		t.Fatal("unselected buttons must stay enabled with Integration")
	}
	if actions[1].Integration.Context["selected_option"] != OptionMaybe {
		t.Fatalf("maybe context: %#v", actions[1].Integration.Context)
	}

	none := EventResponsePostActions(env, "u1", "ev1", "", "http://localhost:8065/plugins/x/action/respond")
	for i, a := range none {
		if a.Style == "primary" || a.Disabled || a.Integration == nil {
			t.Fatalf("unset status: action %d should be clickable: %#v", i, a)
		}
	}

	declined := EventResponsePostActions(env, "u1", "ev1", OptionNo, "http://localhost:8065/plugins/x/action/respond")
	if !declined[0].Disabled || declined[0].Integration != nil {
		t.Fatalf("after decline, Going must be locked without integration: %#v", declined[0])
	}
	if !declined[1].Disabled || declined[1].Integration != nil {
		t.Fatalf("after decline, Maybe must be locked without integration: %#v", declined[1])
	}
	if declined[2].Style != "primary" || !declined[2].Disabled || declined[2].Integration == nil {
		t.Fatalf("after decline, Not going is selected: %#v", declined[2])
	}
}

func TestSanitizeAttachmentLinks(t *testing.T) {
	t.Parallel()
	sa := &model.SlackAttachment{AuthorLink: "mailto:", TitleLink: "not-a-url"}
	SanitizeAttachmentLinks(sa)
	if sa.AuthorLink != "" || sa.TitleLink != "" {
		t.Fatalf("expected cleared links, got author=%q title=%q", sa.AuthorLink, sa.TitleLink)
	}
	sa = &model.SlackAttachment{AuthorLink: "mailto:a@b.c", TitleLink: "https://calendar.yandex.ru/x"}
	SanitizeAttachmentLinks(sa)
	if sa.AuthorLink != "mailto:a@b.c" || sa.TitleLink != "https://calendar.yandex.ru/x" {
		t.Fatalf("valid links must stay: %#v", sa)
	}
}

func TestOptionFromResponseStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{remote.EventResponseStatusAccepted, OptionYes},
		{remote.EventResponseStatusDeclined, OptionNo},
		{remote.EventResponseStatusTentative, OptionMaybe},
		{ResponseMaybe, OptionMaybe},
		{OptionYes, OptionYes},
		{"", ""},
		{remote.EventResponseStatusNotAnswered, ""},
	}
	for _, tc := range cases {
		if got := optionFromResponseStatus(tc.in); got != tc.want {
			t.Fatalf("optionFromResponseStatus(%q)=%q want %q", tc.in, got, tc.want)
		}
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

	if got := ff[FieldResponseStatus].Strings()[0]; got != "Not going" {
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
