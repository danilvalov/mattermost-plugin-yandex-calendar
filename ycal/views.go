package ycal

import (
	"context"
	"fmt"
	"time"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav/caldav"
	"github.com/pkg/errors"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

func (c *client) GetDefaultCalendarView(_ string, start, end time.Time) ([]*remote.Event, error) {
	return c.queryRemoteEvents(start, end)
}

func (c *client) queryRemoteEvents(start, end time.Time) ([]*remote.Event, error) {
	ctx := context.Background()
	cd, calPath, err := c.ensureCalendar(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ycal queryRemoteEvents")
	}

	objs, err := c.queryRaw(ctx, cd, calPath, start, end)
	if err != nil {
		return nil, err
	}

	var out []*remote.Event
	seenByInstance := map[string]int{}
	hasOverrideByInstance := map[string]bool{}
	for _, obj := range objs {
		if obj.Data == nil {
			continue
		}
		for _, comp := range obj.Data.Children {
			if comp.Name != ical.CompEvent {
				continue
			}
			ev, err := veventToRemoteEvent(obj.Data, comp)
			if err != nil || ev == nil || ev.ICalUID == "" {
				continue
			}
			ev.ID = obj.Path
			applyCurrentUserContext(ev, c.email)

			key := recurrenceInstanceKey(obj.Data, comp, ev)
			hasRecurrenceOverride := comp.Props.Get("RECURRENCE-ID") != nil
			out, seenByInstance, hasOverrideByInstance = appendWithRecurrenceDedup(
				out, seenByInstance, hasOverrideByInstance, key, hasRecurrenceOverride, ev,
			)
		}
	}

	return out, nil
}

func appendWithRecurrenceDedup(
	out []*remote.Event,
	seenByInstance map[string]int,
	hasOverrideByInstance map[string]bool,
	key string,
	hasRecurrenceOverride bool,
	ev *remote.Event,
) ([]*remote.Event, map[string]int, map[string]bool) {
	if idx, exists := seenByInstance[key]; exists {
		// If we already have the recurring master instance for this slot and now
		// got a RECURRENCE-ID override, replace master with override.
		if hasRecurrenceOverride && !hasOverrideByInstance[key] {
			out[idx] = ev
			hasOverrideByInstance[key] = true
		}
		return out, seenByInstance, hasOverrideByInstance
	}

	seenByInstance[key] = len(out)
	hasOverrideByInstance[key] = hasRecurrenceOverride
	out = append(out, ev)
	return out, seenByInstance, hasOverrideByInstance
}

func recurrenceInstanceKey(cal *ical.Calendar, ve *ical.Component, ev *remote.Event) string {
	uid := ev.ICalUID
	if uid == "" {
		uid = "<empty-uid>"
	}

	startKey := "<empty-start>"
	if ev.Start != nil {
		startKey = ev.Start.Time().UTC().Format(time.RFC3339Nano)
	}

	rid := ve.Props.Get("RECURRENCE-ID")
	if rid == nil {
		return fmt.Sprintf("%s|%s", uid, startKey)
	}

	loc := time.UTC
	if tzid := rid.Params.Get(ical.ParamTimezoneID); tzid != "" {
		if l := lookupTZ(cal, tzid); l != nil {
			loc = l
		}
	}

	if t, _, _, err := propDateOrDateTime(rid, loc); err == nil {
		startKey = t.UTC().Format(time.RFC3339Nano)
	}

	return fmt.Sprintf("%s|%s", uid, startKey)
}

func (c *client) queryRaw(ctx context.Context, cd *caldav.Client, calPath string, start, end time.Time) ([]caldav.CalendarObject, error) {
	query := &caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{
			Name: "VCALENDAR",
			Comps: []caldav.CalendarCompRequest{{
				Name:     "VEVENT",
				AllProps: true,
				Expand:   &caldav.CalendarExpandRequest{Start: start, End: end},
			}},
		},
		CompFilter: caldav.CompFilter{
			Name: "VCALENDAR",
			Comps: []caldav.CompFilter{{
				Name:  "VEVENT",
				Start: start,
				End:   end,
			}},
		},
	}

	objs, err := cd.QueryCalendar(ctx, calPath, query)
	if err != nil {
		// Retry without expand (some servers reject expand)
		query2 := *query
		query2.CompRequest.Comps[0].Expand = nil
		objs, err = cd.QueryCalendar(ctx, calPath, &query2)
		if err != nil {
			return nil, errors.Wrap(err, "ycal QueryCalendar")
		}
	}
	return objs, nil
}

func (c *client) DoBatchViewCalendarRequests(_ []*remote.ViewCalendarParams) ([]*remote.ViewCalendarResponse, error) {
	return nil, remote.ErrNotImplemented
}
