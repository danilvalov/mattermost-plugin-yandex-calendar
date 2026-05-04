# Yandex CalDAV notes

Official Yandex documentation describes client setup only:

- Server: `https://caldav.yandex.ru`
- Authentication: Yandex account email + **app password** ([sync mobile / CalDAV](https://yandex.com/support/yandex-360/business/calendar/en/sync/sync-mobile))

There is no public spec for non-standard HTTP verbs beyond standard **CalDAV** ([RFC 4791](https://www.rfc-editor.org/rfc/rfc4791)) and **iCalendar** ([RFC 5545](https://www.rfc-editor.org/rfc/rfc5545)).

This plugin implements:

- **Create / update object**: `PUT` with `Content-Type: text/calendar` — same as CalDAV `PutCalendarObject` ([draft-debian-calendar](https://github.com/apple/ccs-calendarserver/blob/master/doc/Extensions/caldav-put.txt) behaviour as implemented by clients).
- **Read**: `GET` on the calendar object resource, or `REPORT` `calendar-query` to locate by `UID`.
- **Accept / decline / tentative**: load the event resource, set `ATTENDEE;PARTSTAT=` for the authenticated user’s `mailto:`, `PUT` the updated calendar (common client behaviour; effectiveness depends on the server accepting in-place updates for invitations).
