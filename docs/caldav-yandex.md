# Yandex CalDAV notes

Official Yandex documentation describes client setup only:

- Server: `https://caldav.yandex.ru`
- Authentication: Yandex account email + **app password** ([sync mobile / CalDAV](https://yandex.com/support/yandex-360/business/calendar/en/sync/sync-mobile))

There is no public spec for non-standard HTTP verbs beyond standard **CalDAV** ([RFC 4791](https://www.rfc-editor.org/rfc/rfc4791)) and **iCalendar** ([RFC 5545](https://www.rfc-editor.org/rfc/rfc5545)).

This plugin implements:

- **Create / update object**: `PUT` with `Content-Type: text/calendar` — same as CalDAV `PutCalendarObject` ([draft-debian-calendar](https://github.com/apple/ccs-calendarserver/blob/master/doc/Extensions/caldav-put.txt) behaviour as implemented by clients).
- **Read**: `GET` on the calendar object resource, or `REPORT` `calendar-query` to locate by `UID`.
- **Accept / decline / tentative**: load the event resource, set `ATTENDEE;PARTSTAT=` for the authenticated user’s `mailto:`, `PUT` the updated calendar (common client behaviour; effectiveness depends on the server accepting in-place updates for invitations).
- **After decline**: Yandex removes (or hides) the attendee copy from CalDAV — subsequent `GET` on the same href returns **404**. There is no documented CalDAV “undecline”; restoring via the Yandex web UI recreates the CalDAV object. The plugin therefore locks **Going** / **Maybe** on the Mattermost post after **Not going**.

## Vendor iCalendar extensions (observed)

Yandex may attach vendor `X-*` properties to `VEVENT` / `ATTENDEE` ([RFC 5545](https://www.rfc-editor.org/rfc/rfc5545) `x-name`). Names are case-insensitive; Yandex emits uppercase.

**Live CalDAV check (attendee copy of an invitation, GET + REPORT, 2026-07):** the object contained standard iCal fields plus **`X-TELEMOST-CONFERENCE`**. It did **not** contain `X-YANDEX-ALLOW-EDIT`, `X-YANDEX-ALLOW-INVITE`, `X-YANDEX-CALENDAR-ID`, `X-YANDEX-UID`, `X-YANDEX-COLOR`, `X-YANDEX-TELEMOST-URL`, or `X-YANDEX-USER-ID` — even when the Yandex web UI had “participants can edit” enabled for that event. Treat catalogues of `X-YANDEX-*` as provisional unless re-verified on a raw `.ics`.

### Participant permissions

| Property / param | Values | Meaning |
| --- | --- | --- |
| `X-YANDEX-ALLOW-EDIT` | `TRUE` / `FALSE` | Guest-edit flag when present (`VEVENT` property or `ATTENDEE` param). Maps to web option **Participants can edit event**. |
| `X-YANDEX-ALLOW-INVITE` | `TRUE` / `FALSE` | Whether attendees may invite others (when present). |

**How this plugin decides UI editability**

1. Cancelled or recurring → read-only (series `PUT` is unsafe in v1).
2. Non-organizer (invitee copy) → read-only for title/time/body. Yandex CalDAV accepts `PUT` with **2xx** then keeps organizer fields (verified 2026-07 on invitees, including events with web UI “participants can edit”). RSVP (`PARTSTAT`) still works.
3. Organizer-owned non-recurring events → editable; `PUT` failures (**403/409**) map to API errors and webapp toast + local rollback.
4. After content `PUT`, the plugin re-`GET`s and returns **403** if requested fields did not stick (guards against silent invitee reverts).

Related product docs: [Editing an event](https://yandex.ru/support/yandex-360/customers/calendar/web/ru/plan-events/events/event-edit), [Calendar settings](https://yandex.ru/support/yandex-360/customers/calendar/web/ru/plan-events/several-calendars/settings).

### Other reported / possible fields

These appear in secondary write-ups; **not seen** on the attendee CalDAV object above:

| Property / param | Values | Notes |
| --- | --- | --- |
| `X-YANDEX-CALENDAR-ID` | numeric | Alleged internal calendar layer id. |
| `X-YANDEX-UID` | hash / UUID | Alleged server-side event id alongside `UID`. |
| `X-YANDEX-COLOR` | HEX | Alleged per-event colour override. |
| `X-YANDEX-TELEMOST-URL` | URL | Alleged Telemost link; live objects used **`X-TELEMOST-CONFERENCE`** instead. |
| `X-YANDEX-USER-ID` (on `ATTENDEE`) | Passport id | Alleged Yandex account id for the attendee. |

### Telemost (video) — observed

| Property | Values | Meaning |
| --- | --- | --- |
| `X-TELEMOST-REQUIRED` | `TRUE` | Client write hint on create/update: Yandex mints a Telemost room. After processing, the object gains `X-TELEMOST-CONFERENCE` and a link line in `DESCRIPTION` (verified 2026-07 via CalDAV `PUT`). |
| `X-TELEMOST-CONFERENCE` | `https://telemost…/j/…` | Direct Telemost room URL (also often duplicated in `DESCRIPTION`). |

See also: [Creating an event](https://yandex.ru/support/yandex-360/customers/calendar/web/ru/plan-events/events/event-create).

### Write / parse caveats

1. **Server-owned rights.** Client-supplied `X-YANDEX-ALLOW-EDIT` on import/`PUT` as a non-organizer is ignored; the server assigns rights from organizer settings ([import](https://yandex.ru/support/yandex-360/customers/calendar/app/ru/import)).
2. **In-place `PUT`.** This plugin patches `SUMMARY` / times / description / `ATTENDEE` list on the existing object, bumps `SEQUENCE` / `LAST-MODIFIED`, drops `METHOD` on update, and leaves unknown `X-*` properties in place. Unchanged `DTSTART`/`DTEND` props are preserved (including `TZID`). Kept attendees retain all existing `ATTENDEE` params (PARTSTAT/ROLE/CUTYPE/X-*/…). Attendee autocomplete uses **Mattermost user search** scoped to the caller’s teams (profile email) — CalDAV has no people directory for the UI.
3. **Guest edit via CalDAV.** Web “participants can edit” does not make invitee content edits persist over CalDAV; the UI therefore treats invitees as read-only for content (RSVP via `PARTSTAT` still works) and links out to Yandex for those changes.
