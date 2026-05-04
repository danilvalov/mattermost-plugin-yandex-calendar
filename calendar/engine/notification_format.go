// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package engine

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/config"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine/views"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/fields"
)

func (processor *notificationProcessor) notifyFieldTitle(mattermostUserID, fieldKey string) string {
	switch fieldKey {
	case FieldSubject:
		return processor.Tr(mattermostUserID, "ycal.notify.field.subject", "Subject", nil)
	case FieldBodyPreview:
		return processor.Tr(mattermostUserID, "ycal.notify.field.body_preview", "BodyPreview", nil)
	case FieldImportance:
		return processor.Tr(mattermostUserID, "ycal.notify.field.importance", "Importance", nil)
	case FieldDuration:
		return processor.Tr(mattermostUserID, "ycal.notify.field.duration", "Duration", nil)
	case FieldWhen:
		return processor.Tr(mattermostUserID, "ycal.notify.field.when", "When", nil)
	case FieldLocation:
		return processor.Tr(mattermostUserID, "ycal.notify.field.location", "Location", nil)
	case FieldAttendees:
		return processor.Tr(mattermostUserID, "ycal.notify.field.attendees", "Attendees", nil)
	case FieldOrganizer:
		return processor.Tr(mattermostUserID, "ycal.notify.field.organizer", "Organizer", nil)
	case FieldResponseStatus:
		return processor.Tr(mattermostUserID, "ycal.notify.field.response_status", "ResponseStatus", nil)
	default:
		return fieldKey
	}
}

func (processor *notificationProcessor) newSlackAttachment(mattermostUserID string, n *remote.Notification) *model.SlackAttachment {
	title := views.EnsureSubject(processor.I18n, mattermostUserID, n.Event.Subject)
	titleLink := n.Event.Weblink
	text := n.Event.BodyPreview
	return &model.SlackAttachment{
		AuthorName: n.Event.Organizer.EmailAddress.Name,
		AuthorLink: "mailto:" + n.Event.Organizer.EmailAddress.Address,
		TitleLink:  titleLink,
		Title:      title,
		Text:       views.MarkdownToHTMLEntities(text),
		Fallback:   fmt.Sprintf("[%s](%s): %s", title, titleLink, views.MarkdownToHTMLEntities(text)),
	}
}

func (processor *notificationProcessor) newEventSlackAttachment(mattermostUserID string, n *remote.Notification, timezone string) *model.SlackAttachment {
	sa := processor.newSlackAttachment(mattermostUserID, n)
	plainTitle := sa.Title
	sa.Title = processor.Tr(mattermostUserID, "ycal.notify.title_new", "(new) {{.Title}}", map[string]any{"Title": plainTitle})

	fields := processor.eventToFields(mattermostUserID, n.Event, timezone)
	for _, k := range notificationFieldOrder {
		v := fields[k]

		sa.Fields = append(sa.Fields, &model.SlackAttachmentField{
			Title: processor.notifyFieldTitle(mattermostUserID, k),
			Value: strings.Join(v.Strings(), ", "),
			Short: true,
		})
	}

	if n.Event.ResponseRequested && !n.Event.IsOrganizer {
		sa.Actions = processor.newPostActionForEventResponse(mattermostUserID, n.Event.ID, n.Event.ResponseStatus.Response, processor.actionURL(config.PathRespond))
	}
	return sa
}

func (processor *notificationProcessor) updatedEventSlackAttachment(mattermostUserID string, n *remote.Notification, prior *remote.Event, timezone string) (bool, *model.SlackAttachment) {
	sa := processor.newSlackAttachment(mattermostUserID, n)
	plainTitle := sa.Title
	sa.Title = processor.Tr(mattermostUserID, "ycal.notify.title_updated", "(updated) {{.Title}}", map[string]any{"Title": plainTitle})

	newFields := processor.eventToFields(mattermostUserID, n.Event, timezone)
	priorFields := processor.eventToFields(mattermostUserID, prior, timezone)
	changed, added, updated, deleted := fields.Diff(priorFields, newFields)
	if !changed {
		return false, nil
	}

	var allChanges []string
	allChanges = append(allChanges, added...)
	allChanges = append(allChanges, updated...)
	allChanges = append(allChanges, deleted...)

	hasImportantChanges := false
	for _, k := range allChanges {
		if isImportantChange(k) {
			hasImportantChanges = true
			break
		}
	}

	if !hasImportantChanges {
		return false, nil
	}

	for _, k := range added {
		if !isImportantChange(k) {
			continue
		}
		sa.Fields = append(sa.Fields, &model.SlackAttachmentField{
			Title: processor.notifyFieldTitle(mattermostUserID, k),
			Value: views.MarkdownToHTMLEntities(strings.Join(newFields[k].Strings(), ", ")),
			Short: true,
		})
	}
	for _, k := range updated {
		if !isImportantChange(k) {
			continue
		}
		sa.Fields = append(sa.Fields, &model.SlackAttachmentField{
			Title: processor.notifyFieldTitle(mattermostUserID, k),
			Value: fmt.Sprintf("~~%s~~ \u2192 %s", views.MarkdownToHTMLEntities(strings.Join(priorFields[k].Strings(), ", ")), views.MarkdownToHTMLEntities(strings.Join(newFields[k].Strings(), ", "))),
			Short: true,
		})
	}
	for _, k := range deleted {
		if !isImportantChange(k) {
			continue
		}
		sa.Fields = append(sa.Fields, &model.SlackAttachmentField{
			Title: processor.notifyFieldTitle(mattermostUserID, k),
			Value: fmt.Sprintf("~~%s~~", views.MarkdownToHTMLEntities(strings.Join(priorFields[k].Strings(), ", "))),
			Short: true,
		})
	}

	if n.Event.ResponseRequested && !n.Event.IsOrganizer && !n.Event.IsCancelled {
		sa.Actions = processor.newPostActionForEventResponse(mattermostUserID, n.Event.ID, n.Event.ResponseStatus.Response, processor.actionURL(config.PathRespond))
	}
	return true, sa
}

func isImportantChange(fieldName string) bool {
	for _, ic := range importantNotificationChanges {
		if ic == fieldName {
			return true
		}
	}
	return false
}

func (processor *notificationProcessor) actionURL(action string) string {
	return fmt.Sprintf("%s%s%s", processor.Config.PluginURLPath, config.PathPostAction, action)
}

func (processor *notificationProcessor) newPostActionForEventResponse(mattermostUserID, eventID, response, url string) []*model.PostAction {
	context := map[string]interface{}{
		config.EventIDKey: eventID,
	}

	pa := &model.PostAction{
		Name: processor.Tr(mattermostUserID, "ycal.notify.response_control", "Response", nil),
		Type: model.PostActionTypeSelect,
		Integration: &model.PostActionIntegration{
			URL:     url,
			Context: context,
		},
	}

	opts := []struct {
		val  string
		text string
		id   string
	}{
		{OptionNotResponded, "Not responded", "ycal.notify.option.not_responded"},
		{OptionYes, "Yes", "ycal.notify.option.yes"},
		{OptionNo, "No", "ycal.notify.option.no"},
		{OptionMaybe, "Maybe", "ycal.notify.option.maybe"},
	}
	for _, o := range opts {
		pa.Options = append(pa.Options, &model.PostActionOptions{
			Text:  processor.Tr(mattermostUserID, o.id, o.text, nil),
			Value: o.val,
		})
	}
	switch response {
	case ResponseNone:
		pa.DefaultOption = OptionNotResponded
	case ResponseYes:
		pa.DefaultOption = OptionYes
	case ResponseNo:
		pa.DefaultOption = OptionNo
	case ResponseMaybe:
		pa.DefaultOption = OptionMaybe
	}
	return []*model.PostAction{pa}
}

func (processor *notificationProcessor) eventToFields(mattermostUserID string, e *remote.Event, timezone string) fields.Fields {
	notDef := processor.Tr(mattermostUserID, "ycal.notify.not_defined", "Not defined", nil)
	noneStr := processor.Tr(mattermostUserID, "ycal.notify.none", "None", nil)
	naStr := processor.Tr(mattermostUserID, "ycal.notify.na", "n/a", nil)
	allDayLabel := processor.Tr(mattermostUserID, "ycal.view.all_day_label", "All day event", nil)

	date := func(dtStart, dtEnd *remote.DateTime, isAllDayEvent bool) (time.Time, time.Time, string) {
		if dtStart == nil || dtEnd == nil {
			return time.Time{}, time.Time{}, naStr
		}

		dtStart = dtStart.In(timezone)
		dtEnd = dtEnd.In(timezone)
		tStart := dtStart.Time()
		tEnd := dtEnd.Time()

		startFormat := "Monday, January 02"
		endDateFormat := "Monday, January 02"

		if isAllDayEvent {
			return tStart, tEnd, tStart.Format(startFormat)+": "+allDayLabel
		}

		if tStart.Year() != time.Now().Year() || tEnd.Year() != time.Now().Year() {
			startFormat += ", 2006"
			endDateFormat += ", 2006"
		}

		startFormat += " · " + time.Kitchen

		var formatted string
		if tStart.Year() != tEnd.Year() || tStart.Month() != tEnd.Month() || tStart.Day() != tEnd.Day() {
			endDateFormat += " · " + time.Kitchen
			formatted = tStart.Format(startFormat) + " - " + tEnd.Format(endDateFormat)
		} else {
			formatted = tStart.Format(startFormat) + " - " + tEnd.Format(time.Kitchen)
		}

		return tStart, tEnd, formatted
	}

	start, end, formattedDate := date(e.Start, e.End, e.IsAllDay)

	minutes := int(end.Sub(start).Round(time.Minute).Minutes())
	hours := int(end.Sub(start).Hours())
	minutes -= hours * 60
	days := int(end.Sub(start).Hours()) / 24
	hours -= days * 24

	dur := ""
	switch {
	case days > 0:
		dur = processor.Tr(mattermostUserID, "ycal.notify.duration_days", "{{.Count}} days", map[string]any{"Count": strconv.Itoa(days)})

	case e.IsAllDay:
		dur = processor.Tr(mattermostUserID, "ycal.notify.duration_allday", "all-day", nil)

	default:
		switch hours {
		case 0:
			// ignore
		case 1:
			dur = processor.Tr(mattermostUserID, "ycal.notify.duration_one_hour", "one hour", nil)
		default:
			if hours > 0 {
				dur = processor.Tr(mattermostUserID, "ycal.notify.duration_hours", "{{.Count}} hours", map[string]any{"Count": strconv.Itoa(hours)})
			}
		}
		if minutes > 0 {
			if dur != "" {
				dur += ", "
			}
			dur += processor.Tr(mattermostUserID, "ycal.notify.duration_minutes", "{{.Count}} minutes", map[string]any{"Count": strconv.Itoa(minutes)})
		}
	}

	attendees := []fields.Value{}
	for _, a := range e.Attendees {
		attendees = append(attendees, fields.NewStringValue(
			fmt.Sprintf("[%s](mailto:%s)",
				a.EmailAddress.Name, a.EmailAddress.Address)))
	}

	if len(attendees) == 0 {
		attendees = append(attendees, fields.NewStringValue(noneStr))
	}

	valOr := func(s string) string {
		if s == "" {
			return notDef
		}
		return s
	}

	ff := fields.Fields{
		FieldSubject:     fields.NewStringValue(views.EnsureSubject(processor.I18n, mattermostUserID, e.Subject)),
		FieldBodyPreview: fields.NewStringValue(views.MarkdownToHTMLEntities(valOr(e.BodyPreview))),
		FieldImportance:  fields.NewStringValue(valOr(e.Importance)),
		FieldWhen:        fields.NewStringValue(valOr(formattedDate)),
		FieldDuration:    fields.NewStringValue(valOr(dur)),
		FieldOrganizer: fields.NewStringValue(
			fmt.Sprintf("[%s](mailto:%s)",
				e.Organizer.EmailAddress.Name, e.Organizer.EmailAddress.Address)),
		FieldLocation:       fields.NewStringValue(views.MarkdownToHTMLEntities(valOr(e.Location.DisplayName))),
		FieldResponseStatus: fields.NewStringValue(e.ResponseStatus.Response),
		FieldAttendees:        fields.NewMultiValue(attendees...),
	}

	return ff
}
