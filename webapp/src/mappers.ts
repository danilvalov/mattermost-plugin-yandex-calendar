import type {SchedulerEvent} from '@mui/x-scheduler/models';

import type {CalendarEventDTO} from './client';
import {
    exclusiveToInclusiveDate,
    inclusiveToExclusiveDate,
    schedulerAllDayToAPI,
    toDateOnly,
} from './dates';

export {exclusiveToInclusiveDate, inclusiveToExclusiveDate, schedulerAllDayToAPI, toDateOnly};

/** Stable CSS class token for event id lookup on click. */
export function eventClassForId(id: string): string {
    let h = 2166136261;
    for (let i = 0; i < id.length; i++) {
        h ^= id.charCodeAt(i);
        h = Math.imul(h, 16777619);
    }
    return `ycal-${(h >>> 0).toString(36)}`;
}

/** MUI only treats trailing Z as an instant; +HH:mm is re-derived via browser local → event TZ (shifts when they differ). */
function toUtcIso(iso: string): string {
    const d = new Date(iso);
    return Number.isNaN(d.getTime()) ? iso : d.toISOString();
}

export function dtoToSchedulerEvent(dto: CalendarEventDTO): SchedulerEvent {
    // MUI uses inclusive all-day end (endOfDay); API/iCal use exclusive end.
    const end = dto.all_day ? exclusiveToInclusiveDate(dto.end) : toUtcIso(dto.end);
    return {
        id: dto.id,
        title: dto.subject || '(no title)',
        description: dto.description,
        start: dto.all_day ? toDateOnly(dto.start) : toUtcIso(dto.start),
        end,
        allDay: dto.all_day,
        timezone: dto.timezone || 'UTC',
        readOnly: !dto.editable,
        draggable: dto.editable,
        resizable: dto.editable,
        className: eventClassForId(dto.id),
    };
}

export function schedulerEventToAPITimes(ev: SchedulerEvent): {start: string; end: string; all_day: boolean} {
    const allDay = Boolean(ev.allDay);
    if (allDay) {
        return {...schedulerAllDayToAPI(ev.start, ev.end), all_day: true};
    }
    return {
        start: String(ev.start),
        end: String(ev.end),
        all_day: false,
    };
}

export function schedulerEventsEqual(a: SchedulerEvent, b: SchedulerEvent): boolean {
    return a.start === b.start && a.end === b.end && Boolean(a.allDay) === Boolean(b.allDay) && a.title === b.title;
}

export function yandexCalendarURL(dto?: CalendarEventDTO | null): string {
    if (dto?.weblink) {
        return dto.weblink;
    }
    return 'https://calendar.yandex.ru/';
}

/** Resolve DTO from a clicked calendar event DOM node (via className token). */
export function dtoFromEventElement(el: Element, dtos: CalendarEventDTO[]): CalendarEventDTO | null {
    let node: Element | null = el;
    while (node) {
        const token = [...node.classList].find((c) => c.startsWith('ycal-'));
        if (token) {
            const hit = dtos.find((d) => eventClassForId(d.id) === token);
            if (hit) {
                return hit;
            }
        }
        node = node.parentElement;
    }
    // Title fallback only when unambiguous (avoid wrong event among same-title items).
    const title = (
        el.querySelector('.MuiEventCalendar-timeGridEventTitle, .MuiEventCalendar-dayGridEventTitle, .MuiEventCalendar-eventItemTitle')?.textContent ||
        el.textContent ||
        ''
    ).trim();
    if (!title) {
        return null;
    }
    const matches = dtos.filter((d) => (d.subject || '(no title)') === title);
    return matches.length === 1 ? matches[0] : null;
}
