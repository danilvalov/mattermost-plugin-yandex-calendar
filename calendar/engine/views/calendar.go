// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package views

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	mmi18n "github.com/mattermost/mattermost/server/public/pluginapi/i18n"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

type Option interface {
	Apply(remote.Event, *model.SlackAttachment)
}

type showTimezoneOption struct {
	timezone string
}

func (tzOpt showTimezoneOption) Apply(event remote.Event, attachment *model.SlackAttachment) {
	attachment.Text = fmt.Sprintf(
		"%s - %s (%s)",
		event.Start.In(tzOpt.timezone).Time().Format(time.Kitchen),
		event.End.In(tzOpt.timezone).Time().Format(time.Kitchen),
		tzOpt.timezone,
	)
}

// ShowTimezoneOption sets attachment text to the event time range in the given timezone.
func ShowTimezoneOption(timezone string) Option {
	if timezone == "" {
		timezone = "UTC"
	}

	return showTimezoneOption{
		timezone: timezone,
	}
}

func RenderCalendarView(events []*remote.Event, timeZone string, bundle *mmi18n.Bundle, mattermostUserID string) (string, error) {
	if len(events) == 0 {
		return tr(bundle, mattermostUserID, "ycal.view.no_upcoming", "You have no upcoming events.", nil), nil
	}

	if timeZone != "" {
		for _, e := range events {
			e.Start = e.Start.In(timeZone)
			e.End = e.End.In(timeZone)
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Start.Time().Before(events[j].Start.Time())
	})

	resp := tr(bundle, mattermostUserID, "ycal.view.times_in_tz", "Times are shown in {{.TimeZone}}",
		map[string]any{"TimeZone": events[0].Start.TimeZone})
	for _, group := range groupEventsByDate(events) {
		resp += "\n" + group[0].Start.Time().Format("Monday January 02, 2006") + "\n\n"
		resp += renderTableHeader(bundle, mattermostUserID)
		for _, e := range group {
			eventString, err := renderEvent(e, true, timeZone, bundle, mattermostUserID)
			if err != nil {
				return "", err
			}
			resp += fmt.Sprintf("\n%s", eventString)
		}
	}

	return resp, nil
}

func RenderDaySummary(events []*remote.Event, timezone string, bundle *mmi18n.Bundle, mattermostUserID string) (string, []*model.SlackAttachment, error) {
	if len(events) == 0 {
		return tr(bundle, mattermostUserID, "ycal.view.no_events_day", "You have no events for that day", nil), nil, nil
	}

	if timezone != "" {
		for _, e := range events {
			e.Start = e.Start.In(timezone)
			e.End = e.End.In(timezone)
		}
	}

	dayLabel := events[0].Start.Time().Format("Monday, 02 January")
	tzLabel := events[0].Start.TimeZone
	message := tr(bundle, mattermostUserID, "ycal.view.agenda_header", "Agenda for {{.Day}}.\nTimes are shown in {{.TimeZone}}",
		map[string]any{"Day": dayLabel, "TimeZone": tzLabel})

	var attachments []*model.SlackAttachment
	for _, event := range events {
		var actions []*model.PostAction

		fields := []*model.SlackAttachmentField{}
		if event.Location != nil && event.Location.DisplayName != "" {
			fields = append(fields, &model.SlackAttachmentField{
				Title: tr(bundle, mattermostUserID, "ycal.view.field.location", "Location", nil),
				Value: MarkdownToHTMLEntities(event.Location.DisplayName),
				Short: true,
			})
		}

		attachments = append(attachments, &model.SlackAttachment{
			Title: event.Subject,
			// Text:    event.BodyPreview,
			Text:    fmt.Sprintf("(%s - %s)", event.Start.In(timezone).Time().Format(time.Kitchen), event.End.In(timezone).Time().Format(time.Kitchen)),
			Fields:  fields,
			Actions: actions,
		})
	}

	return message, attachments, nil
}

func renderTableHeader(bundle *mmi18n.Bundle, mattermostUserID string) string {
	return tr(bundle, mattermostUserID, "ycal.view.table_header", "| Time | Subject |\n| :-- | :-- |", nil)
}

// MarkdownToHTMLEntities converts reserved Markdown characters to their HTML entity equivalents
func MarkdownToHTMLEntities(input string) string {
	replacements := map[rune]string{
		'!':  "&#33;",  // Exclamation Mark
		'#':  "&#35;",  // Hash
		'(':  "&#40;",  // Left Parenthesis
		')':  "&#41;",  // Right Parenthesis
		'*':  "&#42;",  // Asterisk
		'+':  "&#43;",  // Plus Sign
		'-':  "&#45;",  // Dash
		'.':  "&#46;",  // Period
		'/':  "&#47;",  // Forward slash
		':':  "&#58;",  // Colon
		'<':  "&#60;",  // Less than
		'>':  "&#62;",  // Greater Than
		'[':  "&#91;",  // Left Square Bracket
		'\\': "&#92;",  // Back slash
		']':  "&#93;",  // Right Square Bracket
		'_':  "&#95;",  // Underscore
		'`':  "&#96;",  // Backtick
		'|':  "&#124;", // Vertical Bar
		'~':  "&#126;", // Tilde
	}

	var builder strings.Builder
	for _, char := range input {
		if replacement, exists := replacements[char]; exists {
			builder.WriteString(replacement)
		} else {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func renderEvent(event *remote.Event, asRow bool, timeZone string, bundle *mmi18n.Bundle, mattermostUserID string) (string, error) {
	link, err := url.QueryUnescape(event.Weblink)
	if err != nil {
		return "", err
	}

	subject := EnsureSubject(bundle, mattermostUserID, event.Subject)

	if event.IsAllDay {
		allDayLabel := tr(bundle, mattermostUserID, "ycal.view.all_day_label", "All day event", nil)
		if asRow {
			return fmt.Sprintf("| %s | [%s](%s) |", allDayLabel, MarkdownToHTMLEntities(subject), link), nil
		}
		return fmt.Sprintf("(%s) [%s](%s)", allDayLabel, MarkdownToHTMLEntities(subject), link), nil
	}

	start := event.Start.In(timeZone).Time().Format(time.Kitchen)
	end := event.End.In(timeZone).Time().Format(time.Kitchen)

	format := "(%s - %s) [%s](%s)"
	if asRow {
		format = "| %s - %s | [%s](%s) |"
	}

	return fmt.Sprintf(format, start, end, MarkdownToHTMLEntities(subject), link), nil
}

func RenderEventAsAttachment(event *remote.Event, timezone string, bundle *mmi18n.Bundle, mattermostUserID string, options ...Option) (*model.SlackAttachment, error) {
	var actions []*model.PostAction
	fields := []*model.SlackAttachmentField{}
	var titleLink string

	if event.Location != nil && event.Location.DisplayName != "" {
		fields = append(fields, &model.SlackAttachmentField{
			Title: tr(bundle, mattermostUserID, "ycal.view.field.location", "Location", nil),
			Value: MarkdownToHTMLEntities(event.Location.DisplayName),
			Short: true,
		})
	}

	if event.Conference != nil {
		titleLink = event.Conference.URL

		title := tr(bundle, mattermostUserID, "ycal.view.field.meeting_url", "Meeting URL", nil)
		if event.Conference.Application != "" {
			title = event.Conference.Application
		}

		fields = append(fields, &model.SlackAttachmentField{
			Title: title,
			Value: event.Conference.URL,
			Short: true,
		})
	}

	subj := MarkdownToHTMLEntities(EnsureSubject(bundle, mattermostUserID, event.Subject))

	attachment := &model.SlackAttachment{
		Title:     subj,
		TitleLink: titleLink,
		Text:      fmt.Sprintf("%s - %s", event.Start.In(timezone).Time().Format(time.Kitchen), event.End.In(timezone).Time().Format(time.Kitchen)),
		Fields:    fields,
		Actions:   actions,
		Fallback:  fmt.Sprintf("%s\n%s - %s", subj, event.Start.In(timezone).Time().Format(time.Kitchen), event.End.In(timezone).Time().Format(time.Kitchen)),
	}

	for _, opt := range options {
		opt.Apply(*event, attachment)
	}

	return attachment, nil
}

func groupEventsByDate(events []*remote.Event) [][]*remote.Event {
	groups := map[string][]*remote.Event{}

	for _, event := range events {
		date := event.Start.Time().Format("2006-01-02")
		_, ok := groups[date]
		if !ok {
			groups[date] = []*remote.Event{}
		}

		groups[date] = append(groups[date], event)
	}

	days := []string{}
	for k := range groups {
		days = append(days, k)
	}
	sort.Strings(days)

	result := [][]*remote.Event{}
	for _, day := range days {
		group := groups[day]
		result = append(result, group)
	}
	return result
}

func RenderUpcomingEvent(event *remote.Event, timeZone string, bundle *mmi18n.Bundle, mattermostUserID string) (string, error) {
	message := tr(bundle, mattermostUserID, "ycal.view.upcoming_event_prefix", "You have an upcoming event:\n", nil)
	eventString, err := renderEvent(event, false, timeZone, bundle, mattermostUserID)
	if err != nil {
		return "", err
	}

	return message + eventString, nil
}

// EnsureSubject returns a display subject, localized when empty.
func EnsureSubject(bundle *mmi18n.Bundle, mattermostUserID, s string) string {
	if strings.TrimSpace(s) == "" {
		return tr(bundle, mattermostUserID, "ycal.view.no_subject", "(No subject)", nil)
	}

	return s
}

func RenderUpcomingEventAsAttachment(event *remote.Event, timeZone string, bundle *mmi18n.Bundle, mattermostUserID string, options ...Option) (message string, attachment *model.SlackAttachment, err error) {
	message = tr(bundle, mattermostUserID, "ycal.view.upcoming_event_short", "Upcoming event:\n", nil)
	attachment, err = RenderEventAsAttachment(event, timeZone, bundle, mattermostUserID, options...)
	return message, attachment, err
}
