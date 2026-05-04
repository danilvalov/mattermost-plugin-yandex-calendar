// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package ycal

import (
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-ical"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
)

// buildCalendarFromRemoteEvent builds a VCALENDAR with a single VEVENT for PUT to a CalDAV collection.
// Stored calendar objects must not include METHOD (RFC 4791 / iCal interop).
func buildCalendarFromRemoteEvent(in *remote.Event, uid, organizerEmail string) (*ical.Calendar, error) {
	if in.Start == nil || in.End == nil {
		return nil, fmt.Errorf("start and end time are required")
	}
	if uid == "" {
		return nil, fmt.Errorf("uid is required")
	}

	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//Mattermost//Yandex CalDAV EN//")

	ve := ical.NewEvent()
	ve.Props.SetText(ical.PropUID, uid)

	stamp := ical.NewProp(ical.PropDateTimeStamp)
	stamp.SetDateTime(time.Now().UTC())
	ve.Props.Set(stamp)

	if in.IsAllDay {
		start := in.Start.Time()
		end := in.End.Time()
		if start.IsZero() || end.IsZero() {
			return nil, fmt.Errorf("invalid all-day range")
		}
		ds := ical.NewProp(ical.PropDateTimeStart)
		ds.SetDate(start.UTC())
		ve.Props.Set(ds)
		de := ical.NewProp(ical.PropDateTimeEnd)
		de.SetDate(end.UTC())
		ve.Props.Set(de)
	} else {
		start := in.Start.Time()
		end := in.End.Time()
		if start.IsZero() || end.IsZero() {
			return nil, fmt.Errorf("invalid date-time range")
		}
		ds := ical.NewProp(ical.PropDateTimeStart)
		ds.SetDateTime(start)
		ve.Props.Set(ds)
		de := ical.NewProp(ical.PropDateTimeEnd)
		de.SetDateTime(end)
		ve.Props.Set(de)
	}

	if in.Subject != "" {
		ve.Props.SetText(ical.PropSummary, in.Subject)
	}
	if in.Body != nil && in.Body.Content != "" {
		ve.Props.SetText(ical.PropDescription, in.Body.Content)
	}
	if in.Location != nil && in.Location.DisplayName != "" {
		ve.Props.SetText(ical.PropLocation, in.Location.DisplayName)
	}

	if in.ShowAs == showFree {
		ve.Props.SetText(ical.PropTransparency, "TRANSPARENT")
	} else {
		ve.Props.SetText(ical.PropTransparency, "OPAQUE")
	}

	orgMail := organizerEmail
	if in.Organizer != nil && in.Organizer.EmailAddress != nil && in.Organizer.EmailAddress.Address != "" {
		orgMail = in.Organizer.EmailAddress.Address
	}
	org := ical.NewProp(ical.PropOrganizer)
	org.Value = "mailto:" + strings.TrimPrefix(strings.ToLower(orgMail), "mailto:")
	if in.Organizer != nil && in.Organizer.EmailAddress != nil && in.Organizer.EmailAddress.Name != "" {
		org.Params.Set(ical.ParamCommonName, in.Organizer.EmailAddress.Name)
	}
	ve.Props.Set(org)

	for _, a := range in.Attendees {
		if a == nil || a.EmailAddress == nil || a.EmailAddress.Address == "" {
			continue
		}
		ap := ical.NewProp(ical.PropAttendee)
		ap.Params.Set(ical.ParamParticipationStatus, participationFromRemote(a.Status))
		ap.Params.Set(ical.ParamRSVP, "TRUE")
		addr := strings.TrimPrefix(a.EmailAddress.Address, "mailto:")
		ap.Value = "mailto:" + addr
		if a.EmailAddress.Name != "" {
			ap.Params.Set(ical.ParamCommonName, a.EmailAddress.Name)
		}
		ve.Props.Add(ap)
	}

	cal.Children = append(cal.Children, ve.Component)
	return cal, nil
}

func participationFromRemote(rs *remote.EventResponseStatus) string {
	if rs == nil {
		return "NEEDS-ACTION"
	}
	switch rs.Response {
	case remote.EventResponseStatusAccepted:
		return "ACCEPTED"
	case remote.EventResponseStatusDeclined:
		return "DECLINED"
	case remote.EventResponseStatusTentative:
		return "TENTATIVE"
	default:
		return "NEEDS-ACTION"
	}
}

func updateAttendeePartStat(cal *ical.Calendar, userEmail, partStat string) error {
	if cal == nil {
		return fmt.Errorf("nil calendar")
	}
	email := strings.ToLower(strings.TrimSpace(userEmail))
	for _, ch := range cal.Children {
		if ch.Name != ical.CompEvent {
			continue
		}
		key := strings.ToUpper(ical.PropAttendee)
		props := ch.Props[key]
		updated := false
		for i := range props {
			mail := mailtoAddress(props[i].Value)
			if mail != "" && strings.EqualFold(mail, email) {
				props[i].Params.Set(ical.ParamParticipationStatus, partStat)
				updated = true
			}
		}
		if !updated {
			return fmt.Errorf("no ATTENDEE matching user email")
		}
		ch.Props[key] = props
		return nil
	}
	return fmt.Errorf("no VEVENT in calendar")
}

func mailtoAddress(v string) string {
	v = strings.TrimSpace(v)
	if strings.HasPrefix(strings.ToLower(v), "mailto:") {
		return strings.TrimPrefix(v[7:], "")
	}
	return v
}
