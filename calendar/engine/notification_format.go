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
		return processor.Tr(mattermostUserID, "ycal.notify.field.body_preview", "Description", nil)
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
	if strings.TrimSpace(text) == "" && n.Event.Body != nil {
		text = n.Event.Body.Content
	}
	organizerName := ""
	organizerAddress := ""
	if n.Event.Organizer != nil && n.Event.Organizer.EmailAddress != nil {
		organizerName = n.Event.Organizer.EmailAddress.Name
		organizerAddress = n.Event.Organizer.EmailAddress.Address
	}
	return &model.SlackAttachment{
		AuthorName: organizerName,
		AuthorLink: "mailto:" + organizerAddress,
		TitleLink:  titleLink,
		Title:      title,
		Text:       views.LinkifyAndEscapeText(text),
		Fallback:   fmt.Sprintf("[%s](%s): %s", title, titleLink, views.LinkifyAndEscapeText(text)),
	}
}

func (processor *notificationProcessor) newEventSlackAttachment(mattermostUserID string, n *remote.Notification, timezone string, isMilitary bool) *model.SlackAttachment {
	sa := processor.newSlackAttachment(mattermostUserID, n)
	plainTitle := sa.Title
	sa.Title = processor.Tr(mattermostUserID, "ycal.notify.title_new", "(new) {{.Title}}", map[string]any{"Title": plainTitle})

	fields := processor.eventToFields(mattermostUserID, n.Event, timezone, isMilitary)
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

func (processor *notificationProcessor) updatedEventSlackAttachment(mattermostUserID string, n *remote.Notification, prior *remote.Event, timezone string, isMilitary bool) (bool, *model.SlackAttachment) {
	sa := processor.newSlackAttachment(mattermostUserID, n)
	plainTitle := sa.Title
	sa.Title = processor.Tr(mattermostUserID, "ycal.notify.title_updated", "(updated) {{.Title}}", map[string]any{"Title": plainTitle})

	newFields := processor.eventToFields(mattermostUserID, n.Event, timezone, isMilitary)
	priorFields := processor.eventToFields(mattermostUserID, prior, timezone, isMilitary)
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

func (processor *notificationProcessor) eventToFields(mattermostUserID string, e *remote.Event, timezone string, isMilitary bool) fields.Fields {
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

		if isAllDayEvent {
			return tStart, tEnd, processor.formatLocalizedWeekDate(mattermostUserID, tStart, false)+": "+allDayLabel
		}

		includeStartYear := tStart.Year() != time.Now().Year() || tEnd.Year() != time.Now().Year()
		includeEndYear := includeStartYear

		timeLayout := time.Kitchen
		if isMilitary {
			timeLayout = "15:04"
		}
		startDate := processor.formatLocalizedWeekDate(mattermostUserID, tStart, includeStartYear)

		var formatted string
		if tStart.Year() != tEnd.Year() || tStart.Month() != tEnd.Month() || tStart.Day() != tEnd.Day() {
			endDate := processor.formatLocalizedWeekDate(mattermostUserID, tEnd, includeEndYear)
			formatted = startDate + " · " + tStart.Format(timeLayout) + " - " + endDate + " · " + tEnd.Format(timeLayout)
		} else {
			formatted = startDate + " · " + tStart.Format(timeLayout) + " - " + tEnd.Format(timeLayout)
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
		if a == nil || a.EmailAddress == nil {
			continue
		}
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
	organizerName := notDef
	organizerAddress := ""
	if e.Organizer != nil && e.Organizer.EmailAddress != nil {
		organizerName = valOr(e.Organizer.EmailAddress.Name)
		organizerAddress = e.Organizer.EmailAddress.Address
	}

	locationDisplayName := notDef
	if e.Location != nil {
		locationDisplayName = valOr(e.Location.DisplayName)
	}

	ff := fields.Fields{
		FieldSubject:     fields.NewStringValue(views.EnsureSubject(processor.I18n, mattermostUserID, e.Subject)),
		FieldBodyPreview: fields.NewStringValue(views.LinkifyAndEscapeText(valOr(bodyText(e)))),
		FieldImportance:  fields.NewStringValue(valOr(e.Importance)),
		FieldWhen:        fields.NewStringValue(valOr(formattedDate)),
		FieldDuration:    fields.NewStringValue(valOr(dur)),
		FieldOrganizer: fields.NewStringValue(
			fmt.Sprintf("[%s](mailto:%s)",
				organizerName, organizerAddress)),
		FieldLocation:       fields.NewStringValue(views.MarkdownToHTMLEntities(locationDisplayName)),
		FieldResponseStatus: fields.NewStringValue(processor.localizeResponseStatus(mattermostUserID, e.ResponseStatus)),
		FieldAttendees:      fields.NewMultiValue(attendees...),
	}

	return ff
}

func (processor *notificationProcessor) localizedWeekday(mattermostUserID string, weekday time.Weekday) string {
	switch weekday {
	case time.Monday:
		return processor.Tr(mattermostUserID, "ycal.notify.date.weekday.monday", "Monday", nil)
	case time.Tuesday:
		return processor.Tr(mattermostUserID, "ycal.notify.date.weekday.tuesday", "Tuesday", nil)
	case time.Wednesday:
		return processor.Tr(mattermostUserID, "ycal.notify.date.weekday.wednesday", "Wednesday", nil)
	case time.Thursday:
		return processor.Tr(mattermostUserID, "ycal.notify.date.weekday.thursday", "Thursday", nil)
	case time.Friday:
		return processor.Tr(mattermostUserID, "ycal.notify.date.weekday.friday", "Friday", nil)
	case time.Saturday:
		return processor.Tr(mattermostUserID, "ycal.notify.date.weekday.saturday", "Saturday", nil)
	case time.Sunday:
		return processor.Tr(mattermostUserID, "ycal.notify.date.weekday.sunday", "Sunday", nil)
	default:
		return ""
	}
}

func (processor *notificationProcessor) localizedMonth(mattermostUserID string, month time.Month) string {
	switch month {
	case time.January:
		return processor.Tr(mattermostUserID, "ycal.notify.date.month.january", "January", nil)
	case time.February:
		return processor.Tr(mattermostUserID, "ycal.notify.date.month.february", "February", nil)
	case time.March:
		return processor.Tr(mattermostUserID, "ycal.notify.date.month.march", "March", nil)
	case time.April:
		return processor.Tr(mattermostUserID, "ycal.notify.date.month.april", "April", nil)
	case time.May:
		return processor.Tr(mattermostUserID, "ycal.notify.date.month.may", "May", nil)
	case time.June:
		return processor.Tr(mattermostUserID, "ycal.notify.date.month.june", "June", nil)
	case time.July:
		return processor.Tr(mattermostUserID, "ycal.notify.date.month.july", "July", nil)
	case time.August:
		return processor.Tr(mattermostUserID, "ycal.notify.date.month.august", "August", nil)
	case time.September:
		return processor.Tr(mattermostUserID, "ycal.notify.date.month.september", "September", nil)
	case time.October:
		return processor.Tr(mattermostUserID, "ycal.notify.date.month.october", "October", nil)
	case time.November:
		return processor.Tr(mattermostUserID, "ycal.notify.date.month.november", "November", nil)
	case time.December:
		return processor.Tr(mattermostUserID, "ycal.notify.date.month.december", "December", nil)
	default:
		return ""
	}
}

func (processor *notificationProcessor) formatLocalizedWeekDate(mattermostUserID string, t time.Time, includeYear bool) string {
	data := map[string]any{
		"Weekday": processor.localizedWeekday(mattermostUserID, t.Weekday()),
		"Day":     fmt.Sprintf("%02d", t.Day()),
		"Month":   processor.localizedMonth(mattermostUserID, t.Month()),
		"Year":    strconv.Itoa(t.Year()),
	}

	if includeYear {
		return processor.Tr(mattermostUserID, "ycal.notify.date.format_with_year", "{{.Weekday}} {{.Month}} {{.Day}}, {{.Year}}", data)
	}

	return processor.Tr(mattermostUserID, "ycal.notify.date.format_without_year", "{{.Weekday}} {{.Month}} {{.Day}}", data)
}

func bodyText(e *remote.Event) string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.BodyPreview) != "" {
		return e.BodyPreview
	}
	if e.Body != nil {
		return e.Body.Content
	}
	return ""
}

func (processor *notificationProcessor) localizeResponseStatus(mattermostUserID string, status *remote.EventResponseStatus) string {
	if status == nil {
		return processor.Tr(mattermostUserID, "ycal.notify.option.not_responded", "Not responded", nil)
	}

	switch status.Response {
	case remote.EventResponseStatusAccepted:
		return processor.Tr(mattermostUserID, "ycal.notify.option.yes", "Yes", nil)
	case remote.EventResponseStatusDeclined:
		return processor.Tr(mattermostUserID, "ycal.notify.option.no", "No", nil)
	case remote.EventResponseStatusTentative:
		return processor.Tr(mattermostUserID, "ycal.notify.option.maybe", "Maybe", nil)
	default:
		return processor.Tr(mattermostUserID, "ycal.notify.option.not_responded", "Not responded", nil)
	}
}
