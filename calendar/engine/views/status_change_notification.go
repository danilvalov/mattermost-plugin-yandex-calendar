// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package views

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	mmi18n "github.com/mattermost/mattermost/server/public/pluginapi/i18n"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func prettyStatus(bundle *mmi18n.Bundle, mattermostUserID, status string) string {
	switch status {
	case model.StatusOnline:
		return tr(bundle, mattermostUserID, "ycal.status.online", "Online", nil)
	case model.StatusAway:
		return tr(bundle, mattermostUserID, "ycal.status.away", "Away", nil)
	case model.StatusDnd:
		return tr(bundle, mattermostUserID, "ycal.status.dnd", "Do Not Disturb", nil)
	case model.StatusOffline:
		return tr(bundle, mattermostUserID, "ycal.status.offline", "Offline", nil)
	default:
		return status
	}
}

func RenderStatusChangeNotificationView(events []*remote.Event, status, url string, bundle *mmi18n.Bundle, mattermostUserID string) *model.SlackAttachment {
	for _, e := range events {
		if e.Start.Time().After(time.Now()) {
			return statusChangeAttachments(e, status, url, bundle, mattermostUserID)
		}
	}

	nEvents := len(events)
	if nEvents > 0 && status == model.StatusDnd {
		return statusChangeAttachments(events[nEvents-1], status, url, bundle, mattermostUserID)
	}

	return statusChangeAttachments(nil, status, url, bundle, mattermostUserID)
}

func RenderEventWillStartLine(bundle *mmi18n.Bundle, mattermostUserID, subject, weblink string, startTime time.Time) string {
	link, _ := url.QueryUnescape(weblink)
	data := map[string]any{"Subject": subject, "Link": link}
	var eventString string
	if startTime.Before(time.Now()) {
		if subject == "" {
			eventString = tr(bundle, mattermostUserID, "ycal.status.event_ongoing_no_subject", "[An event with no subject]({{.Link}}) is ongoing.", map[string]any{"Link": link})
		} else {
			eventString = tr(bundle, mattermostUserID, "ycal.status.event_ongoing", "Your event [{{.Subject}}]({{.Link}}) is ongoing.", data)
		}
	} else {
		if subject == "" {
			eventString = tr(bundle, mattermostUserID, "ycal.status.event_will_start_no_subject", "[An event with no subject]({{.Link}}) will start soon.", map[string]any{"Link": link})
		} else {
			eventString = tr(bundle, mattermostUserID, "ycal.status.event_will_start", "Your event [{{.Subject}}]({{.Link}}) will start soon.", data)
		}
	}
	return eventString
}

func renderScheduleItem(event *remote.Event, status string, bundle *mmi18n.Bundle, mattermostUserID string) string {
	if event == nil {
		return tr(bundle, mattermostUserID, "ycal.status.no_upcoming_change_back",
			"You have no upcoming events.\n Shall I change your status back to {{.Status}}?",
			map[string]any{"Status": prettyStatus(bundle, mattermostUserID, status)})
	}

	resp := RenderEventWillStartLine(bundle, mattermostUserID, event.Subject, event.Weblink, event.Start.Time())

	resp += "\n" + tr(bundle, mattermostUserID, "ycal.status.change_status_prompt",
		"Shall I change your status to {{.Status}}?",
		map[string]any{"Status": prettyStatus(bundle, mattermostUserID, status)})
	return resp
}

func statusChangeAttachments(event *remote.Event, status, url string, bundle *mmi18n.Bundle, mattermostUserID string) *model.SlackAttachment {
	pretty := prettyStatus(bundle, mattermostUserID, status)

	actionYes := &model.PostAction{
		Name: tr(bundle, mattermostUserID, "ycal.common.yes", "Yes", nil),
		Integration: &model.PostActionIntegration{
			URL: url,
			Context: map[string]interface{}{
				"value":            true,
				"change_to":        status,
				"pretty_change_to": pretty,
				"hasEvent":         false,
			},
		},
	}

	actionNo := &model.PostAction{
		Name: tr(bundle, mattermostUserID, "ycal.common.no", "No", nil),
		Integration: &model.PostActionIntegration{
			URL: url,
			Context: map[string]interface{}{
				"value":    false,
				"hasEvent": false,
			},
		},
	}

	if event != nil {
		marshalledStart, _ := json.Marshal(event.Start.Time())
		actionYes.Integration.Context["hasEvent"] = true
		actionYes.Integration.Context["subject"] = event.Subject
		actionYes.Integration.Context["weblink"] = event.Weblink
		actionYes.Integration.Context["startTime"] = string(marshalledStart)

		actionNo.Integration.Context["hasEvent"] = true
		actionNo.Integration.Context["subject"] = event.Subject
		actionNo.Integration.Context["weblink"] = event.Weblink
		actionNo.Integration.Context["startTime"] = string(marshalledStart)
	}

	title := tr(bundle, mattermostUserID, "ycal.status.change_title", "Status change", nil)
	text := renderScheduleItem(event, status, bundle, mattermostUserID)
	sa := &model.SlackAttachment{
		Title:    title,
		Text:     text,
		Actions:  []*model.PostAction{actionYes, actionNo},
		Fallback: fmt.Sprintf("%s: %s", title, text),
	}

	return sa
}
