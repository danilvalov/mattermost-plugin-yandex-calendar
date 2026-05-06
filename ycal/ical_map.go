package ycal

import (
	"fmt"
	"strings"
	"time"

	"github.com/danilvalov/mattermost-plugin-yandex-calendar/calendar/remote"
	"github.com/emersion/go-ical"
)

const (
	showBusy = "busy"
	showFree = "free"
)

// veventToRemoteEvent maps an iCal VEVENT (with optional VTIMEZONE siblings) to remote.Event.
func veventToRemoteEvent(parent *ical.Calendar, ve *ical.Component) (*remote.Event, error) {
	if ve.Name != ical.CompEvent {
		return nil, fmt.Errorf("not a VEVENT")
	}

	uid, err := ve.Props.Text(ical.PropUID)
	if err != nil {
		uid = ""
	}

	summary, _ := ve.Props.Text(ical.PropSummary)
	desc, _ := ve.Props.Text(ical.PropDescription)
	location, _ := ve.Props.Text(ical.PropLocation)
	status, _ := ve.Props.Text(ical.PropStatus)
	url := propText(ve, ical.PropURL, "URL")

	start, end, isAllDay, err := veventDateTimes(parent, ve)
	if err != nil {
		return nil, err
	}

	showAs := showBusy
	if transp, _ := ve.Props.Text(ical.PropTransparency); strings.EqualFold(transp, "TRANSPARENT") {
		showAs = showFree
	}

	org := &remote.Attendee{}
	if o := ve.Props.Get(ical.PropOrganizer); o != nil {
		mail := organizerMail(o)
		org.EmailAddress = &remote.EmailAddress{
			Address: mail,
			Name:    attendeeDisplayName(mail, organizerCN(o)),
		}
	}

	resp := &remote.EventResponseStatus{Response: remote.EventResponseStatusNotAnswered}

	ev := &remote.Event{
		ID:             uid,
		ICalUID:        uid,
		Subject:        summary,
		Body:           &remote.ItemBody{Content: desc},
		BodyPreview:    clip(desc, 512),
		IsAllDay:       isAllDay,
		ShowAs:         showAs,
		Weblink:        strings.TrimSpace(url),
		Start:          start,
		End:            end,
		Location:       locationRemote(location),
		IsCancelled:    strings.EqualFold(status, "CANCELLED"),
		IsOrganizer:    false,
		Organizer:      org,
		ResponseStatus: resp,
		Attendees:      attendeesFromVEVENT(ve),
	}

	return ev, nil
}

func applyCurrentUserContext(ev *remote.Event, userEmail string) {
	if ev == nil {
		return
	}

	normUser := normalizeEmail(userEmail)
	if normUser == "" {
		return
	}

	if ev.Organizer != nil && ev.Organizer.EmailAddress != nil {
		ev.IsOrganizer = normalizeEmail(ev.Organizer.EmailAddress.Address) == normUser
	}

	for _, a := range ev.Attendees {
		if a == nil || a.EmailAddress == nil || a.Status == nil {
			continue
		}
		if normalizeEmail(a.EmailAddress.Address) == normUser {
			ev.ResponseStatus = &remote.EventResponseStatus{Response: a.Status.Response}
			return
		}
	}
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func locationRemote(loc string) *remote.Location {
	if loc == "" {
		return nil
	}
	return &remote.Location{DisplayName: loc}
}

func organizerMail(o *ical.Prop) string {
	if o == nil {
		return ""
	}
	v := o.Value
	if strings.HasPrefix(strings.ToLower(v), "mailto:") {
		return v[7:]
	}
	return v
}

func normalizeEmail(v string) string {
	return strings.TrimSpace(strings.ToLower(v))
}

func propText(ve *ical.Component, names ...string) string {
	for _, name := range names {
		if name == "" {
			continue
		}
		if p := ve.Props.Get(name); p != nil {
			return strings.TrimSpace(p.Value)
		}
	}
	return ""
}

func organizerCN(o *ical.Prop) string {
	if o == nil {
		return ""
	}
	if cn := o.Params.Get(ical.ParamCommonName); cn != "" {
		return cn
	}
	return ""
}

func attendeesFromVEVENT(ve *ical.Component) []*remote.Attendee {
	var out []*remote.Attendee
	for _, att := range ve.Props.Values(ical.PropAttendee) {
		addr := att.Value
		if strings.HasPrefix(strings.ToLower(addr), "mailto:") {
			addr = addr[7:]
		}
		cn := att.Params.Get(ical.ParamCommonName)
		partstat := att.Params.Get(ical.ParamParticipationStatus)
		rs := mapPartStat(partstat)
		out = append(out, &remote.Attendee{
			EmailAddress: &remote.EmailAddress{
				Address: addr,
				Name:    attendeeDisplayName(addr, cn),
			},
			Status: &remote.EventResponseStatus{Response: rs},
		})
	}
	return out
}

// attendeeDisplayName supplies EmailAddress.Name for Mattermost markdown
// ("[Name](mailto:addr)"). iCal often omits CN; use local-part of email as fallback.
func attendeeDisplayName(mailaddr, cn string) string {
	if cn != "" {
		return cn
	}
	if mailaddr == "" {
		return ""
	}
	if i := strings.LastIndex(mailaddr, "@"); i > 0 {
		return mailaddr[:i]
	}
	return mailaddr
}

func mapPartStat(ps string) string {
	switch strings.ToUpper(ps) {
	case "ACCEPTED":
		return remote.EventResponseStatusAccepted
	case "DECLINED":
		return remote.EventResponseStatusDeclined
	case "TENTATIVE":
		return remote.EventResponseStatusTentative
	default:
		return remote.EventResponseStatusNotAnswered
	}
}

func veventDateTimes(cal *ical.Calendar, ve *ical.Component) (start *remote.DateTime, end *remote.DateTime, allDay bool, err error) {
	dtStart := ve.Props.Get(ical.PropDateTimeStart)
	dtEnd := ve.Props.Get(ical.PropDateTimeEnd)
	if dtStart == nil {
		return nil, nil, false, fmt.Errorf("missing DTSTART")
	}

	loc := time.UTC
	if tzid := dtStart.Params.Get(ical.ParamTimezoneID); tzid != "" {
		if l := lookupTZ(cal, tzid); l != nil {
			loc = l
		}
	}

	t0, dateOnly0, utcStart, err := propDateOrDateTime(dtStart, loc)
	if err != nil {
		return nil, nil, false, err
	}

	var t1 time.Time
	var dateOnly1 bool
	if dtEnd != nil {
		l2 := loc
		if tzid := dtEnd.Params.Get(ical.ParamTimezoneID); tzid != "" {
			if l := lookupTZ(cal, tzid); l != nil {
				l2 = l
			}
		}
		t1, dateOnly1, _, err = propDateOrDateTime(dtEnd, l2)
		if err != nil {
			return nil, nil, false, err
		}
	} else {
		if dateOnly0 {
			t1 = t0.Add(24 * time.Hour)
			dateOnly1 = true
		} else {
			t1 = t0.Add(time.Hour)
		}
	}

	allDay = dateOnly0 || dateOnly1
	if allDay {
		return remote.NewDateTime(t0.UTC(), "UTC"), remote.NewDateTime(t1.UTC(), "UTC"), true, nil
	}

	if utcStart {
		loc = time.UTC
	}
	tzName := loc.String()
	if tzName == "Local" {
		tzName = "UTC"
	}
	return remote.NewDateTime(t0, tzName), remote.NewDateTime(t1, tzName), false, nil
}

func propDateOrDateTime(p *ical.Prop, fallbackLoc *time.Location) (t time.Time, dateOnly bool, explicitUTC bool, err error) {
	if p == nil {
		return time.Time{}, false, false, fmt.Errorf("nil prop")
	}
	if p.Params.Get(ical.ParamValue) == string(ical.ValueDate) || len(p.Value) == 8 {
		layout := "20060102"
		v := strings.TrimSpace(p.Value)
		if strings.Contains(v, "-") {
			layout = "2006-01-02"
		}
		t, err = time.ParseInLocation(layout, v, fallbackLoc)
		if err != nil {
			return time.Time{}, false, false, err
		}
		return t, true, false, nil
	}
	v := strings.TrimSpace(p.Value)
	if strings.HasSuffix(v, "Z") {
		if len(v) >= 16 && v[8] == 'T' {
			t, err = time.Parse("20060102T150405Z", strings.TrimSuffix(v, "Z")+"Z")
		} else {
			t, err = time.Parse(time.RFC3339, v)
		}
		return t.UTC(), false, true, err
	}
	if len(v) >= 15 && v[8] == 'T' {
		t, err = time.ParseInLocation("20060102T150405", v[:15], fallbackLoc)
		return t, false, false, err
	}
	t, err = time.ParseInLocation("2006-01-02T15:04:05", v, fallbackLoc)
	return t, false, false, err
}

func lookupTZ(cal *ical.Calendar, tzid string) *time.Location {
	if loc, err := time.LoadLocation(tzid); err == nil {
		return loc
	}

	if cal == nil {
		return nil
	}
	for _, ch := range cal.Children {
		if ch.Name != ical.CompTimezone {
			continue
		}
		tzidProp, _ := ch.Props.Text(ical.PropTimezoneID)
		if tzidProp != tzid {
			continue
		}
		if loc := timezoneFromComponent(ch); loc != nil {
			return loc
		}
	}
	return nil
}

func timezoneFromComponent(vtz *ical.Component) *time.Location {
	for _, ch := range vtz.Children {
		if ch.Name != ical.CompTimezoneStandard && ch.Name != ical.CompTimezoneDaylight {
			continue
		}
		if tzoffset := ch.Props.Get(ical.PropTimezoneOffsetTo); tzoffset != nil {
			off, err := parseTzOffset(tzoffset.Value)
			if err == nil {
				return time.FixedZone("ics", off)
			}
		}
	}
	return nil
}

func parseTzOffset(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	sign := 1
	if s[0] == '-' {
		sign = -1
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}
	if len(s) < 4 {
		return 0, fmt.Errorf("short")
	}
	h := int(s[0]-'0')*10 + int(s[1]-'0')
	m := int(s[2]-'0')*10 + int(s[3]-'0')
	sec := (h*3600 + m*60) * sign
	return sec, nil
}
