// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/pkg/errors"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine/views"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/bot"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/httputils"
)

const (
	createEventDateTimeFormat = "2006-01-02 15:04"
	createEventDateFormat     = "2006-01-02"
)

type createEventPayload struct {
	AllDay    bool     `json:"all_day"`
	Attendees []string `json:"attendees"`
	Date      string   `json:"date"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	// ISO fields for calendar product UI (multi-day + past allowed).
	StartISO    string `json:"start"`
	EndISO      string `json:"end"`
	Description string `json:"description,omitempty"`
	Subject     string `json:"subject"`
	Location    string `json:"location,omitempty"`
	ChannelID   string `json:"channel_id"`
}

func (cep createEventPayload) isISO() bool {
	return strings.TrimSpace(cep.StartISO) != "" && strings.TrimSpace(cep.EndISO) != ""
}

func (cep createEventPayload) ToRemoteEvent(loc *time.Location) (*remote.Event, error) {
	var evt remote.Event

	evt.IsAllDay = cep.AllDay
	evt.Subject = cep.Subject
	if cep.Description != "" {
		evt.Body = &remote.ItemBody{
			Content:     cep.Description,
			ContentType: "text/plain",
		}
	}
	if cep.Location != "" {
		evt.Location = &remote.Location{
			DisplayName: cep.Location,
		}
	}

	if cep.isISO() {
		start, startDate, err := parseRFC3339OrDate(cep.StartISO, loc)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing start")
		}
		end, endDate, err := parseRFC3339OrDate(cep.EndISO, loc)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing end")
		}
		allDay := cep.AllDay || (startDate && endDate)
		evt.IsAllDay = allDay
		tzName := loc.String()
		if allDay {
			startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
			endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
			if !endDay.After(startDay) {
				return nil, fmt.Errorf("end must be after start")
			}
			evt.Start = remote.NewDateTime(startDay, "UTC")
			evt.End = remote.NewDateTime(endDay, "UTC")
		} else {
			if !end.After(start) {
				return nil, fmt.Errorf("end must be after start")
			}
			evt.Start = remote.NewDateTime(start, tzName)
			evt.End = remote.NewDateTime(end, tzName)
		}
		return &evt, nil
	}

	if !cep.AllDay {
		start, err := cep.parseStartTime(loc)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing start time")
		}

		end, err := cep.parseEndTime(loc)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing start time")
		}

		evt.Start = &remote.DateTime{
			DateTime: start.Format(remote.RFC3339NanoNoTimezone),
			TimeZone: loc.String(),
		}
		evt.End = &remote.DateTime{
			DateTime: end.Format(remote.RFC3339NanoNoTimezone),
			TimeZone: loc.String(),
		}
	} else {
		date, err := cep.parseDate(loc)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing date")
		}

		startDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		endDay := startDay.Add(24 * time.Hour)
		evt.Start = remote.NewDateTime(startDay, "UTC")
		evt.End = remote.NewDateTime(endDay, "UTC")
	}

	return &evt, nil
}

func (cep createEventPayload) parseStartTime(loc *time.Location) (time.Time, error) {
	return time.ParseInLocation(createEventDateTimeFormat, fmt.Sprintf("%s %s", cep.Date, cep.StartTime), loc)
}

func (cep createEventPayload) parseEndTime(loc *time.Location) (time.Time, error) {
	return time.ParseInLocation(createEventDateTimeFormat, fmt.Sprintf("%s %s", cep.Date, cep.EndTime), loc)
}

func (cep createEventPayload) parseDate(loc *time.Location) (time.Time, error) {
	return time.ParseInLocation(createEventDateFormat, cep.Date, loc)
}

func (cep createEventPayload) IsValid(loc *time.Location) error {
	if cep.Subject == "" {
		return fmt.Errorf("subject must not be empty")
	}

	if cep.isISO() {
		start, startDate, err := parseRFC3339OrDate(cep.StartISO, loc)
		if err != nil {
			return fmt.Errorf("please use a valid start time")
		}
		end, endDate, err := parseRFC3339OrDate(cep.EndISO, loc)
		if err != nil {
			return fmt.Errorf("please use a valid end time")
		}
		allDay := cep.AllDay || (startDate && endDate)
		if allDay {
			startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
			endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
			if !endDay.After(startDay) {
				return fmt.Errorf("end date cannot be earlier than start date")
			}
			return nil
		}
		if !end.After(start) {
			return fmt.Errorf("end date cannot be earlier than start date")
		}
		return nil
	}

	if cep.Date == "" {
		return fmt.Errorf("date must not be empty")
	}

	_, err := cep.parseDate(loc)
	if err != nil {
		return fmt.Errorf("invalid date")
	}

	if cep.StartTime == "" && cep.EndTime == "" && !cep.AllDay {
		return fmt.Errorf("start time/end time must be set or event should last all day")
	}

	start, err := cep.parseStartTime(loc)
	if err != nil {
		return fmt.Errorf("please use a valid start time")
	}

	if start.Before(time.Now()) {
		return fmt.Errorf("please select a start date and time that is not prior to the current time")
	}

	end, err := cep.parseEndTime(loc)
	if err != nil {
		return fmt.Errorf("please use a valid end time")
	}

	if end.Before(time.Now()) {
		return fmt.Errorf("please select an end date and time that is not prior to the current time")
	}

	if start.After(end) {
		return fmt.Errorf("end date cannot be earlier than start date")
	}

	return nil
}

func (api *api) createEvent(w http.ResponseWriter, r *http.Request) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" {
		api.Logger.Errorf("createEvent, unauthorized user")
		httputils.WriteUnauthorizedError(w, fmt.Errorf("unauthorized"))
		return
	}

	user, errStore := api.Store.LoadUser(mattermostUserID)
	if errStore != nil && !errors.Is(errStore, store.ErrNotFound) {
		api.Logger.With(bot.LogContext{"err": errStore}).Errorf("createEvent, error occurred while loading user from store")
		httputils.WriteInternalServerError(w, errStore)
		return
	}
	if errors.Is(errStore, store.ErrNotFound) {
		api.Logger.With(bot.LogContext{"err": errStore.Error()}).Errorf("createEvent, user not found in store")
		httputils.WriteUnauthorizedError(w, fmt.Errorf("unauthorized"))
		return
	}

	var payload createEventPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		api.Logger.With(bot.LogContext{"err": err.Error()}).Errorf("createEvent, error occurred while decoding event payload")
		httputils.WriteBadRequestError(w, err)
		return
	}
	defer r.Body.Close()

	if payload.ChannelID != "" {
		if !api.PluginAPI.CanLinkEventToChannel(payload.ChannelID, user.MattermostUserID) {
			api.Logger.With(bot.LogContext{"userID": mattermostUserID, "channelID": payload.ChannelID}).Errorf("createEvent, user don't have permission to link events in the selected channel")
			httputils.WriteBadRequestError(w, fmt.Errorf("you don't have permission to link events in the selected channel"))
			return
		}
	}

	client := api.Remote.MakeUserClient(context.Background(), user.OAuth2Token, mattermostUserID, api.Poster, api.Store)

	mailbox, errMailbox := client.GetMailboxSettings(user.Remote.ID)
	if errMailbox != nil {
		api.Logger.With(bot.LogContext{"err": errMailbox.Error(), "userID": mattermostUserID}).Errorf("createEvent, error occurred while getting mailbox settings for user")
		httputils.WriteInternalServerError(w, errMailbox)
		return
	}

	timezone := mailbox.TimeZone
	if timezone == "" {
		timezone = "UTC"
	}
	mattermostUser, errMMUser := api.PluginAPI.GetMattermostUser(mattermostUserID)
	if errMMUser == nil && mattermostUser != nil && mattermostUser.Timezone != nil {
		if mattermostUser.Timezone["useAutomaticTimezone"] == "true" {
			if tz := mattermostUser.Timezone["automaticTimezone"]; tz != "" {
				timezone = tz
			}
		} else if tz := mattermostUser.Timezone["manualTimezone"]; tz != "" {
			timezone = tz
		}
	}

	loc, errLocation := time.LoadLocation(timezone)
	if errLocation != nil {
		api.Logger.With(bot.LogContext{"err": errLocation.Error(), "timezone": timezone}).Errorf("createEvent, error occurred while loading mailbox timezone location")
		httputils.WriteInternalServerError(w, errLocation)
		return
	}

	if err := payload.IsValid(loc); err != nil {
		api.Logger.Errorf("createEvent, invalid payload")
		httputils.WriteBadRequestError(w, err)
		return
	}

	event, errParse := payload.ToRemoteEvent(loc)
	if errParse != nil {
		api.Logger.With(bot.LogContext{"err": errParse.Error()}).Errorf("createEvent, error occurred while creating remote event from payload")
		httputils.WriteBadRequestError(w, errParse)
		return
	}

	atts, resolveErr := api.resolveAttendeeEmails(payload.Attendees)
	if resolveErr != nil {
		httputils.WriteBadRequestError(w, resolveErr)
		return
	}
	event.Attendees = atts

	event, err := client.CreateEvent(event)
	if err != nil {
		api.Logger.With(bot.LogContext{"err": err.Error()}).Errorf("createEvent, error occurred while creating event")
		httputils.WriteInternalServerError(w, err)
		return
	}

	isMilitary := false
	pref, prefErr := api.PluginAPI.GetPreferenceForUser(mattermostUserID, "display", "use_military_time")
	if prefErr != nil || pref == nil {
		pref, prefErr = api.PluginAPI.GetPreferenceForUser(mattermostUserID, "display_settings", "use_military_time")
	}
	if prefErr != nil {
		prefs, allErr := api.PluginAPI.GetPreferencesForUser(mattermostUserID)
		if allErr == nil {
			for _, p := range prefs {
				if (p.Category == "display" || p.Category == "display_settings") && p.Name == "use_military_time" {
					isMilitary = p.Value == "true"
					break
				}
			}
		}
	} else if pref != nil {
		isMilitary = pref.Value == "true"
	}
	attachment, err := views.RenderEventAsAttachmentWithTimeFormat(event, timezone, isMilitary, api.I18n, mattermostUserID, views.ShowTimezoneOptionWithTimeFormat(timezone, isMilitary))
	if err != nil {
		api.Logger.With(bot.LogContext{"err": err.Error()}).Errorf("createEvent, error rendering event as attachment")
	}

	// Event linking
	if payload.ChannelID != "" {
		if err := api.Store.StoreUserLinkedEvent(user.MattermostUserID, event.ICalUID, payload.ChannelID); err != nil {
			api.Poster.DM(mattermostUserID, api.Tr(mattermostUserID, "ycal.api.event_link_failed",
				"Your event **{{.Subject}}** could not be linked to a channel. Please contact an administrator for more details.",
				map[string]any{"Subject": event.Subject}))
			api.Logger.With(bot.LogContext{"err": err.Error(), "userID": user.MattermostUserID}).Errorf("createEvent, error occurred while storing user linked event")
			httputils.WriteInternalServerError(w, err)
			return
		}

		if err := api.Store.AddLinkedChannelToEvent(event.ICalUID, payload.ChannelID); err != nil {
			api.Logger.With(bot.LogContext{"err": err}).Errorf("error linking event to channel")
			defer func() {
				api.Poster.DM(mattermostUserID, api.Tr(mattermostUserID, "ycal.api.event_link_failed",
					"Your event **{{.Subject}}** could not be linked to a channel. Please contact an administrator for more details.",
					map[string]any{"Subject": event.Subject}))
			}()
		} else {
			post := &model.Post{
				Message: api.Tr(mattermostUserID, "ycal.api.event_linked_channel",
					"The event **{{.Subject}}** was linked to this channel by @{{.User}}",
					map[string]any{"Subject": event.Subject, "User": user.MattermostUsername}),
				ChannelId: payload.ChannelID,
			}
			if attachment != nil {
				model.ParseSlackAttachment(post, []*model.SlackAttachment{attachment})
			}
			if err := api.Poster.CreatePost(post); err != nil {
				api.Logger.With(bot.LogContext{"err": err}).Errorf("error sending post to channel about linked event")
			}
		}
	} else {
		if attachment == nil {
			api.Poster.DM(mattermostUserID, api.Tr(mattermostUserID, "ycal.api.event_created_with_subject",
				"Your event: **{{.Subject}}** was created successfully.", map[string]any{"Subject": event.Subject}))
		} else {
			api.Poster.DMWithMessageAndAttachments(mattermostUserID, api.Tr(mattermostUserID, "ycal.api.event_created",
				"Your event was created successfully.", nil), attachment)
		}
	}

	httputils.WriteJSONResponse(w, eventToDTO(event), http.StatusCreated)
}
