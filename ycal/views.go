package ycal

import (
	"context"
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
			if ev.ID == "" {
				ev.ID = obj.Path
			}
			out = append(out, ev)
		}
	}

	return out, nil
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
