package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/engine"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/store"
	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/utils/httputils"
)

type calendarEventDTO struct {
	ID                 string               `json:"id"`
	ICalUID            string               `json:"ical_uid"`
	Subject            string               `json:"subject"`
	Description        string               `json:"description,omitempty"`
	Location           string               `json:"location,omitempty"`
	Start              string               `json:"start"`
	End                string               `json:"end"`
	AllDay             bool                 `json:"all_day"`
	Timezone           string               `json:"timezone"`
	Editable           bool                 `json:"editable"`
	IsOrganizer        bool                 `json:"is_organizer"`
	IsRecurring        bool                 `json:"is_recurring"`
	IsCancelled        bool                 `json:"is_cancelled"`
	ResponseRequested  bool                 `json:"response_requested"`
	ResponseStatus     string               `json:"response_status,omitempty"`
	Weblink            string               `json:"weblink,omitempty"`
	ConferenceURL      string               `json:"conference_url,omitempty"`
	Attendees          []calendarAttendeeDTO `json:"attendees,omitempty"`
}

type calendarAttendeeDTO struct {
	Name   string `json:"name,omitempty"`
	Email  string `json:"email"`
	Status string `json:"status,omitempty"`
}

func eventToDTO(ev *remote.Event) calendarEventDTO {
	if ev == nil {
		return calendarEventDTO{}
	}
	dto := calendarEventDTO{
		ID:                ev.ID,
		ICalUID:           ev.ICalUID,
		Subject:           ev.Subject,
		AllDay:            ev.IsAllDay,
		Editable:          ev.Editable(),
		IsOrganizer:       ev.IsOrganizer,
		IsRecurring:       ev.IsRecurring,
		IsCancelled:       ev.IsCancelled,
		ResponseRequested: ev.ResponseRequested,
		Weblink:           ev.Weblink,
	}
	if ev.Conference != nil {
		dto.ConferenceURL = ev.Conference.URL
	}
	if ev.Body != nil {
		dto.Description = ev.Body.Content
	}
	if ev.Location != nil {
		dto.Location = ev.Location.DisplayName
	}
	if ev.ResponseStatus != nil {
		dto.ResponseStatus = ev.ResponseStatus.Response
	}
	if ev.Start != nil {
		dto.Timezone = ev.Start.TimeZone
		if ev.IsAllDay {
			dto.Start = ev.Start.Time().UTC().Format("2006-01-02")
		} else {
			// Always UTC Z so MUI Scheduler takes the instant path (non-Z offsets are
			// re-parsed as wall-time via browser local → event TZ and shift the board).
			dto.Start = ev.Start.Time().UTC().Format(time.RFC3339)
		}
	}
	if ev.End != nil {
		if ev.IsAllDay {
			dto.End = ev.End.Time().UTC().Format("2006-01-02")
		} else {
			dto.End = ev.End.Time().UTC().Format(time.RFC3339)
		}
		if dto.Timezone == "" {
			dto.Timezone = ev.End.TimeZone
		}
	}
	if dto.Timezone == "" {
		dto.Timezone = "UTC"
	}
	for _, a := range ev.Attendees {
		if a == nil || a.EmailAddress == nil || a.EmailAddress.Address == "" {
			continue
		}
		ad := calendarAttendeeDTO{
			Name:  a.EmailAddress.Name,
			Email: a.EmailAddress.Address,
		}
		if a.Status != nil {
			ad.Status = a.Status.Response
		}
		dto.Attendees = append(dto.Attendees, ad)
	}
	return dto
}

func (api *api) loadConnectedUser(r *http.Request) (*store.User, error) {
	mattermostUserID := r.Header.Get("Mattermost-User-Id")
	if mattermostUserID == "" {
		return nil, errUnauthorized
	}
	user, err := api.Store.LoadUser(mattermostUserID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, errUnauthorized
		}
		return nil, err
	}
	return user, nil
}

var (
	errUnauthorized = errors.New("unauthorized")
)

func (api *api) writeStoreOrAuthError(w http.ResponseWriter, err error) {
	if errors.Is(err, errUnauthorized) {
		httputils.WriteUnauthorizedError(w, err)
		return
	}
	httputils.WriteInternalServerError(w, err)
}

func (api *api) writeEngineError(w http.ResponseWriter, err error) {
	var statusErr *remote.StatusError
	if errors.As(err, &statusErr) && statusErr != nil {
		msg := errors.New(statusErr.Error())
		switch {
		case statusErr.Code == http.StatusForbidden:
			httputils.WriteForbiddenError(w, msg)
		case statusErr.Code == http.StatusConflict || statusErr.Code == http.StatusPreconditionFailed:
			httputils.WriteJSONError(w, http.StatusConflict, "Conflict.", msg)
		case statusErr.Code == http.StatusNotFound:
			httputils.WriteNotFoundError(w, msg)
		case statusErr.Code >= 400 && statusErr.Code < 500:
			httputils.WriteJSONError(w, statusErr.Code, "Request rejected by calendar server.", msg)
		default:
			httputils.WriteInternalServerError(w, msg)
		}
		return
	}
	if errors.Is(err, engine.ErrForbidden) {
		httputils.WriteForbiddenError(w, err)
		return
	}
	if errors.Is(err, engine.ErrBadRequest) {
		httputils.WriteBadRequestError(w, err)
		return
	}
	msg := err.Error()
	if strings.Contains(strings.ToLower(msg), "not found") || strings.Contains(msg, "no event") {
		httputils.WriteNotFoundError(w, err)
		return
	}
	httputils.WriteInternalServerError(w, err)
}

func mattermostUserTimezone(api *api, mattermostUserID, fallback string) string {
	timezone := fallback
	if timezone == "" {
		timezone = "UTC"
	}
	mattermostUser, err := api.PluginAPI.GetMattermostUser(mattermostUserID)
	if err != nil || mattermostUser == nil || mattermostUser.Timezone == nil {
		return timezone
	}
	if mattermostUser.Timezone["useAutomaticTimezone"] == "true" {
		if tz := mattermostUser.Timezone["automaticTimezone"]; tz != "" {
			return tz
		}
	} else if tz := mattermostUser.Timezone["manualTimezone"]; tz != "" {
		return tz
	}
	return timezone
}

func parseRFC3339OrDate(s string, loc *time.Location) (time.Time, bool, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false, fmt.Errorf("empty time")
	}
	if loc == nil {
		loc = time.UTC
	}
	if len(s) == 10 && s[4] == '-' && s[7] == '-' {
		t, err := time.ParseInLocation("2006-01-02", s, loc)
		return t, true, err
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.In(loc), false, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.In(loc), false, nil
	}
	// MUI Scheduler emits wall-clock ISO without offset; interpret in user location.
	for _, layout := range []string{
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
	} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, false, nil
		}
	}
	return time.Time{}, false, fmt.Errorf("invalid time %q", s)
}
