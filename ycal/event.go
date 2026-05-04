// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package ycal

import (
	"context"
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
		return nil, errors.Wrap(err, "ycal PutCalendarObject")
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
	return errors.Wrap(err, "ycal PutCalendarObject")
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
