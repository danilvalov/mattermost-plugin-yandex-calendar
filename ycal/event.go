// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package ycal

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav/caldav"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func (c *client) GetEvent(_ string, eventID string) (*remote.Event, error) {
	ctx := context.Background()
	_, objPath, err := c.loadCalendarObject(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if objPath.Data == nil {
		return nil, errors.New("ycal: empty calendar object")
	}
	for _, comp := range objPath.Data.Children {
		if comp.Name != ical.CompEvent {
			continue
		}
		ev, err := veventToRemoteEvent(objPath.Data, comp)
		if err != nil || ev == nil {
			continue
		}
		ev.ID = objPath.Path
		if ev.ICalUID == "" {
			ev.ICalUID = ev.ID
		}
		applyCurrentUserContext(ev, c.email)
		return ev, nil
	}
	return nil, errors.New("ycal: no VEVENT in object")
}

func (c *client) CreateEvent(in *remote.Event) (*remote.Event, error) {
	ctx := context.Background()
	cd, calPath, err := c.ensureCalendar(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ycal CreateEvent")
	}

	uid := in.ICalUID
	if uid == "" {
		uid = uuid.New().String()
	}

	cal, err := buildCalendarFromRemoteEvent(in, uid, c.email)
	if err != nil {
		return nil, err
	}

	fileName := safeICSFileName(uid)
	objPath := joinCalendarObjectPath(calPath, fileName)

	co, err := cd.PutCalendarObject(ctx, objPath, cal)
	if err != nil {
		return nil, wrapCalDAVWriteErr("create", errors.Wrap(err, "ycal PutCalendarObject"))
	}

	out, err := c.GetEvent("", co.Path)
	if err != nil {
		return nil, err
	}
	if out != nil && out.ICalUID == "" {
		out.ICalUID = uid
	}
	return out, nil
}

func (c *client) UpdateEvent(in *remote.Event) (*remote.Event, error) {
	if in == nil || strings.TrimSpace(in.ID) == "" {
		return nil, errors.New("ycal UpdateEvent: event id required")
	}
	ctx := context.Background()
	cd, obj, err := c.loadCalendarObject(ctx, in.ID)
	if err != nil {
		return nil, errors.Wrap(err, "ycal UpdateEvent")
	}
	if obj.Data == nil {
		return nil, errors.New("ycal UpdateEvent: empty calendar object")
	}

	ve := firstVEVENT(obj.Data)
	if ve == nil {
		return nil, errors.New("ycal UpdateEvent: no VEVENT")
	}
	if err := patchVEVENTFromRemote(obj.Data, ve, in); err != nil {
		return nil, err
	}
	// Stored objects should not carry METHOD (RFC 4791); Yandex sometimes serves METHOD:PUBLISH on invites.
	obj.Data.Props.Del(ical.PropMethod)

	if _, err := cd.PutCalendarObject(ctx, obj.Path, obj.Data); err != nil {
		return nil, wrapCalDAVWriteErr("update", errors.Wrap(err, "ycal UpdateEvent PutCalendarObject"))
	}
	out, err := c.GetEvent("", obj.Path)
	if err != nil {
		return nil, err
	}
	if err := assertEventUpdateApplied(in, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *client) DeleteEvent(eventID string) error {
	ctx := context.Background()
	cd, obj, err := c.loadCalendarObject(ctx, eventID)
	if err != nil {
		return errors.Wrap(err, "ycal DeleteEvent")
	}
	if err := cd.DeleteCalendarObject(ctx, obj.Path); err != nil {
		return wrapCalDAVWriteErr("delete", errors.Wrap(err, "ycal DeleteCalendarObject"))
	}
	return nil
}

func firstVEVENT(cal *ical.Calendar) *ical.Component {
	if cal == nil {
		return nil
	}
	for _, ch := range cal.Children {
		if ch.Name == ical.CompEvent {
			return ch
		}
	}
	return nil
}

// patchVEVENTFromRemote updates DTSTART/DTEND/SUMMARY/DESCRIPTION/LOCATION/DTSTAMP in-place.
func patchVEVENTFromRemote(cal *ical.Calendar, ve *ical.Component, in *remote.Event) error {
	if in.Start == nil || in.End == nil {
		return errors.New("ycal: start and end required")
	}

	now := time.Now().UTC()
	stamp := ical.NewProp(ical.PropDateTimeStamp)
	stamp.SetDateTime(now)
	ve.Props.Set(stamp)
	lm := ical.NewProp(ical.PropLastModified)
	lm.SetDateTime(now)
	ve.Props.Set(lm)
	bumpSequence(ve)

	if timesNeedRewrite(cal, ve, in) {
		if in.IsAllDay {
			start := in.Start.Time().UTC()
			end := in.End.Time().UTC()
			startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
			endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
			if !endDay.After(startDay) {
				endDay = startDay.Add(24 * time.Hour)
			}
			ds := ical.NewProp(ical.PropDateTimeStart)
			ds.SetDate(startDay)
			ve.Props.Set(ds)
			de := ical.NewProp(ical.PropDateTimeEnd)
			de.SetDate(endDay)
			ve.Props.Set(de)
		} else {
			start := in.Start.Time()
			end := in.End.Time()
			if start.IsZero() || end.IsZero() {
				return errors.New("ycal: invalid date-time range")
			}
			loc := eventTimeLocation(ve, start.Location())
			ds := ical.NewProp(ical.PropDateTimeStart)
			ds.SetDateTime(start.In(loc))
			ve.Props.Set(ds)
			de := ical.NewProp(ical.PropDateTimeEnd)
			de.SetDateTime(end.In(loc))
			ve.Props.Set(de)
		}
	}

	if in.Subject != "" {
		ve.Props.SetText(ical.PropSummary, in.Subject)
	}
	if in.Body != nil {
		if in.Body.Content == "" {
			ve.Props.Del(ical.PropDescription)
		} else {
			ve.Props.SetText(ical.PropDescription, in.Body.Content)
		}
	}
	if in.Location != nil {
		if in.Location.DisplayName == "" {
			ve.Props.Del(ical.PropLocation)
		} else {
			ve.Props.SetText(ical.PropLocation, in.Location.DisplayName)
		}
	}

	return nil
}

// timesNeedRewrite is false when wall-clock start/end already match (keeps original TZID props).
func timesNeedRewrite(cal *ical.Calendar, ve *ical.Component, in *remote.Event) bool {
	curStart, curEnd, allDay, err := veventDateTimes(cal, ve)
	if err != nil || curStart == nil || curEnd == nil {
		return true
	}
	if allDay != in.IsAllDay {
		return true
	}
	wantStart, wantEnd := in.Start.Time(), in.End.Time()
	if wantStart.IsZero() || wantEnd.IsZero() {
		return true
	}
	if allDay {
		cs, ce := curStart.Time().UTC(), curEnd.Time().UTC()
		ws, we := wantStart.UTC(), wantEnd.UTC()
		return cs.Year() != ws.Year() || cs.Month() != ws.Month() || cs.Day() != ws.Day() ||
			ce.Year() != we.Year() || ce.Month() != we.Month() || ce.Day() != we.Day()
	}
	return !curStart.Time().Truncate(time.Second).Equal(wantStart.Truncate(time.Second)) ||
		!curEnd.Time().Truncate(time.Second).Equal(wantEnd.Truncate(time.Second))
}

func eventTimeLocation(ve *ical.Component, fallback *time.Location) *time.Location {
	if ve != nil {
		if ds := ve.Props.Get(ical.PropDateTimeStart); ds != nil {
			if tzid := ds.Params.Get(ical.ParamTimezoneID); tzid != "" {
				if loc, err := time.LoadLocation(tzid); err == nil {
					return loc
				}
			}
		}
	}
	if fallback != nil {
		return fallback
	}
	return time.UTC
}

func bumpSequence(ve *ical.Component) {
	seq := 0
	if p := ve.Props.Get(ical.PropSequence); p != nil {
		seq, _ = strconv.Atoi(strings.TrimSpace(p.Value))
	}
	ve.Props.SetText(ical.PropSequence, strconv.Itoa(seq+1))
	if p := ve.Props.Get("X-MICROSOFT-CDO-APPT-SEQUENCE"); p != nil {
		ms, _ := strconv.Atoi(strings.TrimSpace(p.Value))
		ve.Props.SetText("X-MICROSOFT-CDO-APPT-SEQUENCE", strconv.Itoa(ms+1))
	}
}

// assertEventUpdateApplied catches Yandex accepting PUT then silently reverting invitee edits.
func assertEventUpdateApplied(want, got *remote.Event) error {
	if want == nil || got == nil {
		return nil
	}
	denied := remote.NewStatusError(http.StatusForbidden,
		"Yandex Calendar did not apply the update (no permission to edit this event)", nil)
	if want.Subject != "" && got.Subject != want.Subject {
		return denied
	}
	if want.Location != nil && want.Location.DisplayName != "" {
		gotLoc := ""
		if got.Location != nil {
			gotLoc = got.Location.DisplayName
		}
		if gotLoc != want.Location.DisplayName {
			return denied
		}
	}
	if want.Start != nil && got.Start != nil && !want.IsAllDay {
		ws, gs := want.Start.Time().Truncate(time.Second), got.Start.Time().Truncate(time.Second)
		if !ws.IsZero() && !gs.IsZero() && !ws.Equal(gs) {
			return denied
		}
	}
	if want.End != nil && got.End != nil && !want.IsAllDay {
		we, ge := want.End.Time().Truncate(time.Second), got.End.Time().Truncate(time.Second)
		if !we.IsZero() && !ge.IsZero() && !we.Equal(ge) {
			return denied
		}
	}
	return nil
}

func (c *client) AcceptEvent(_, eventID string) error {
	return c.updateOwnParticipation(eventID, "ACCEPTED")
}

func (c *client) DeclineEvent(_, eventID string) error {
	return c.updateOwnParticipation(eventID, "DECLINED")
}

func (c *client) TentativelyAcceptEvent(eventID string) error {
	return c.updateOwnParticipation(eventID, "TENTATIVE")
}

func (c *client) updateOwnParticipation(eventID, partStat string) error {
	ctx := context.Background()
	cd, obj, err := c.loadCalendarObject(ctx, eventID)
	if err != nil {
		return err
	}
	if err := updateAttendeePartStat(obj.Data, c.email, partStat); err != nil {
		return errors.Wrap(err, "ycal update participation")
	}
	_, err = cd.PutCalendarObject(ctx, obj.Path, obj.Data)
	return wrapCalDAVWriteErr("update", errors.Wrap(err, "ycal PutCalendarObject"))
}

func (c *client) GetEventsBetweenDates(_ string, start, end time.Time) ([]*remote.Event, error) {
	return c.queryRemoteEvents(start, end)
}

// loadCalendarObject resolves eventID to a path (full CalDAV path or UID), then GETs the resource.
func (c *client) loadCalendarObject(ctx context.Context, eventID string) (*caldav.Client, *caldav.CalendarObject, error) {
	cd, calPath, err := c.ensureCalendar(ctx)
	if err != nil {
		return nil, nil, err
	}

	pathOrUID := strings.TrimSpace(eventID)
	if pathOrUID == "" {
		return nil, nil, errors.New("ycal: empty event id")
	}

	var objPath string
	if isAbsoluteCalPath(pathOrUID) {
		objPath = pathOrUID
	} else {
		objPath, err = c.findObjectPathByUID(ctx, cd, calPath, pathOrUID)
		if err != nil {
			return nil, nil, err
		}
	}

	obj, err := cd.GetCalendarObject(ctx, objPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "ycal GetCalendarObject")
	}
	return cd, obj, nil
}

func (c *client) findObjectPathByUID(ctx context.Context, cd *caldav.Client, calPath, uid string) (string, error) {
	start := time.Now().Add(-730 * 24 * time.Hour)
	end := time.Now().Add(730 * 24 * time.Hour)
	query := &caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{
			Name: "VCALENDAR",
			Comps: []caldav.CalendarCompRequest{{
				Name:     "VEVENT",
				AllProps: true,
			}},
		},
		CompFilter: caldav.CompFilter{
			Name: "VCALENDAR",
			Comps: []caldav.CompFilter{{
				Name:  "VEVENT",
				Start: start,
				End:   end,
				Props: []caldav.PropFilter{{
					Name: ical.PropUID,
					TextMatch: &caldav.TextMatch{
						Text:            uid,
						NegateCondition: false,
					},
				}},
			}},
		},
	}

	objs, err := cd.QueryCalendar(ctx, calPath, query)
	if err != nil {
		return "", errors.Wrap(err, "ycal query by UID")
	}
	if len(objs) == 0 {
		// Some servers ignore UID TextMatch; scan a wide range client-side.
		return c.scanCalendarForUID(ctx, cd, calPath, uid)
	}
	return objs[0].Path, nil
}

func (c *client) scanCalendarForUID(ctx context.Context, cd *caldav.Client, calPath, uid string) (string, error) {
	start := time.Now().Add(-730 * 24 * time.Hour)
	end := time.Now().Add(730 * 24 * time.Hour)
	objs, err := c.queryRaw(ctx, cd, calPath, start, end)
	if err != nil {
		return "", err
	}
	for _, obj := range objs {
		if obj.Data == nil {
			continue
		}
		for _, ch := range obj.Data.Children {
			if ch.Name != ical.CompEvent {
				continue
			}
			u, _ := ch.Props.Text(ical.PropUID)
			if u == uid {
				return obj.Path, nil
			}
		}
	}
	return "", errors.Errorf("ycal: no event with UID %q", uid)
}
