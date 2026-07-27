package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/bot"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/httputils"
)

func (api *api) listEvents(w http.ResponseWriter, r *http.Request) {
	user, err := api.loadConnectedUser(r)
	if err != nil {
		api.writeStoreOrAuthError(w, err)
		return
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		httputils.WriteBadRequestError(w, fmt.Errorf("from and to query params are required"))
		return
	}
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		httputils.WriteBadRequestError(w, fmt.Errorf("invalid from: %w", err))
		return
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		httputils.WriteBadRequestError(w, fmt.Errorf("invalid to: %w", err))
		return
	}
	if !to.After(from) {
		httputils.WriteBadRequestError(w, fmt.Errorf("to must be after from"))
		return
	}

	eng := engine.New(api.Env, user.MattermostUserID)
	events, err := eng.ViewCalendar(engine.NewUser(user.MattermostUserID), from, to)
	if err != nil {
		api.Logger.With(bot.LogContext{"err": err.Error()}).Errorf("listEvents")
		api.writeEngineError(w, err)
		return
	}

	out := make([]calendarEventDTO, 0, len(events))
	for _, ev := range events {
		out = append(out, eventToDTO(ev))
	}
	_ = httputils.WriteJSONResponse(w, out, http.StatusOK)
}

func (api *api) getEvent(w http.ResponseWriter, r *http.Request) {
	user, err := api.loadConnectedUser(r)
	if err != nil {
		api.writeStoreOrAuthError(w, err)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		httputils.WriteBadRequestError(w, fmt.Errorf("id is required"))
		return
	}
	eng := engine.New(api.Env, user.MattermostUserID)
	ev, err := eng.GetEvent(engine.NewUser(user.MattermostUserID), id)
	if err != nil {
		api.writeEngineError(w, err)
		return
	}
	_ = httputils.WriteJSONResponse(w, eventToDTO(ev), http.StatusOK)
}

type patchEventPayload struct {
	ID          string    `json:"id"`
	Subject     *string   `json:"subject,omitempty"`
	Start       *string   `json:"start,omitempty"`
	End         *string   `json:"end,omitempty"`
	AllDay      *bool     `json:"all_day,omitempty"`
	Description *string   `json:"description,omitempty"`
	Location    *string   `json:"location,omitempty"`
	Attendees   *[]string `json:"attendees,omitempty"`
}

func (api *api) patchEvent(w http.ResponseWriter, r *http.Request) {
	user, err := api.loadConnectedUser(r)
	if err != nil {
		api.writeStoreOrAuthError(w, err)
		return
	}
	var payload patchEventPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputils.WriteBadRequestError(w, err)
		return
	}
	defer r.Body.Close()
	if payload.ID == "" {
		httputils.WriteBadRequestError(w, fmt.Errorf("id is required"))
		return
	}

	eng := engine.New(api.Env, user.MattermostUserID)
	existing, err := eng.GetEvent(engine.NewUser(user.MattermostUserID), payload.ID)
	if err != nil {
		api.writeEngineError(w, err)
		return
	}

	tz := mattermostUserTimezone(api, user.MattermostUserID, "")
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
		tz = "UTC"
	}

	updated := *existing
	updated.ID = payload.ID
	if payload.Subject != nil {
		updated.Subject = *payload.Subject
	}
	if payload.Description != nil {
		updated.Body = &remote.ItemBody{Content: *payload.Description, ContentType: "text/plain"}
	}
	if payload.Location != nil {
		updated.Location = &remote.Location{DisplayName: *payload.Location}
	}
	if payload.AllDay != nil {
		updated.IsAllDay = *payload.AllDay
	}

	if payload.Start != nil || payload.End != nil {
		startStr := ""
		endStr := ""
		if existing.Start != nil {
			if existing.IsAllDay {
				startStr = existing.Start.Time().UTC().Format("2006-01-02")
			} else {
				startStr = existing.Start.Time().Format(time.RFC3339)
			}
		}
		if existing.End != nil {
			if existing.IsAllDay {
				endStr = existing.End.Time().UTC().Format("2006-01-02")
			} else {
				endStr = existing.End.Time().Format(time.RFC3339)
			}
		}
		if payload.Start != nil {
			startStr = *payload.Start
		}
		if payload.End != nil {
			endStr = *payload.End
		}
		start, startDate, err := parseRFC3339OrDate(startStr, loc)
		if err != nil {
			httputils.WriteBadRequestError(w, err)
			return
		}
		end, endDate, err := parseRFC3339OrDate(endStr, loc)
		if err != nil {
			httputils.WriteBadRequestError(w, err)
			return
		}
		allDay := updated.IsAllDay || (startDate && endDate)
		updated.IsAllDay = allDay
		if allDay {
			startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
			endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
			if !endDay.After(startDay) {
				httputils.WriteBadRequestError(w, fmt.Errorf("end must be after start"))
				return
			}
			updated.Start = remote.NewDateTime(startDay, "UTC")
			updated.End = remote.NewDateTime(endDay, "UTC")
		} else {
			if !end.After(start) {
				httputils.WriteBadRequestError(w, fmt.Errorf("end must be after start"))
				return
			}
			updated.Start = remote.NewDateTime(start, tz)
			updated.End = remote.NewDateTime(end, tz)
		}
	}

	if payload.Attendees != nil {
		atts, resolveErr := api.resolveAttendeeEmails(*payload.Attendees)
		if resolveErr != nil {
			httputils.WriteBadRequestError(w, resolveErr)
			return
		}
		updated.Attendees = atts
		updated.RewriteAttendees = true
	}

	out, err := eng.UpdateEvent(engine.NewUser(user.MattermostUserID), &updated)
	if err != nil {
		api.Logger.With(bot.LogContext{"err": err.Error()}).Errorf("patchEvent")
		api.writeEngineError(w, err)
		return
	}
	_ = httputils.WriteJSONResponse(w, eventToDTO(out), http.StatusOK)
}

func (api *api) deleteEvent(w http.ResponseWriter, r *http.Request) {
	user, err := api.loadConnectedUser(r)
	if err != nil {
		api.writeStoreOrAuthError(w, err)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		httputils.WriteBadRequestError(w, fmt.Errorf("id is required"))
		return
	}
	eng := engine.New(api.Env, user.MattermostUserID)
	if err := eng.DeleteEvent(engine.NewUser(user.MattermostUserID), id); err != nil {
		api.writeEngineError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type respondEventPayload struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (api *api) respondEvent(w http.ResponseWriter, r *http.Request) {
	user, err := api.loadConnectedUser(r)
	if err != nil {
		api.writeStoreOrAuthError(w, err)
		return
	}
	var payload respondEventPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httputils.WriteBadRequestError(w, err)
		return
	}
	defer r.Body.Close()
	if payload.ID == "" {
		httputils.WriteBadRequestError(w, fmt.Errorf("id is required"))
		return
	}

	eng := engine.New(api.Env, user.MattermostUserID)
	u := engine.NewUser(user.MattermostUserID)
	var respErr error
	switch payload.Status {
	case remote.EventResponseStatusAccepted:
		respErr = eng.AcceptEvent(u, payload.ID)
	case remote.EventResponseStatusDeclined:
		respErr = eng.DeclineEvent(u, payload.ID)
	case remote.EventResponseStatusTentative:
		respErr = eng.TentativelyAcceptEvent(u, payload.ID)
	default:
		httputils.WriteBadRequestError(w, fmt.Errorf("invalid status"))
		return
	}
	if respErr != nil {
		api.writeEngineError(w, respErr)
		return
	}
	ev, err := eng.GetEvent(u, payload.ID)
	if err != nil {
		// Decline often 404s the attendee copy; Accept/Tentative must keep the event in UI.
		msg := strings.ToLower(err.Error())
		if payload.Status == remote.EventResponseStatusDeclined &&
			(strings.Contains(msg, "not found") || strings.Contains(msg, "no event") || strings.Contains(msg, "404")) {
			_ = httputils.WriteJSONResponse(w, map[string]any{"ok": true}, http.StatusOK)
			return
		}
		api.writeEngineError(w, err)
		return
	}
	_ = httputils.WriteJSONResponse(w, eventToDTO(ev), http.StatusOK)
}

// resolveAttendeeEmails maps emails or Mattermost user IDs to remote.Attendee (deduped).
func (api *api) resolveAttendeeEmails(raw []string) ([]*remote.Attendee, error) {
	seen := map[string]struct{}{}
	out := make([]*remote.Attendee, 0, len(raw))
	for _, pa := range raw {
		pa = strings.TrimSpace(pa)
		if pa == "" {
			continue
		}
		email := ""
		name := ""
		if strings.Contains(pa, "@") {
			email = pa
		} else {
			mmUser, err := api.PluginAPI.GetMattermostUser(pa)
			if err != nil {
				return nil, fmt.Errorf("unknown attendee %q", pa)
			}
			email = mmUser.Email
			name = strings.TrimSpace(mmUser.GetFullName())
			if name == "" {
				name = mmUser.Username
			}
		}
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" || !strings.Contains(email, "@") {
			return nil, fmt.Errorf("attendee %q has no email", pa)
		}
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}
		out = append(out, &remote.Attendee{
			EmailAddress: &remote.EmailAddress{Address: email, Name: name},
		})
	}
	return out, nil
}
