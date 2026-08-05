// ponytail: timed events with non-Z offsets must become UTC Z so MUI takes the instant path.
const assert = require('assert');

function toUtcIso(iso) {
    const d = new Date(iso);
    return Number.isNaN(d.getTime()) ? iso : d.toISOString();
}

function dtoToSchedulerEvent(dto) {
    const end = dto.all_day ? dto.end : toUtcIso(dto.end);
    return {
        start: dto.all_day ? dto.start : toUtcIso(dto.start),
        end,
        allDay: Boolean(dto.all_day),
        timezone: dto.timezone || 'UTC',
    };
}

// prod bug: 15:00+06 == 09:00Z == 12:00 Europe/Kirov; must not stay as +06:00 for MUI
assert.strictEqual(toUtcIso('2026-08-03T15:00:00+06:00'), '2026-08-03T09:00:00.000Z');
assert.strictEqual(toUtcIso('2026-08-03T12:00:00+03:00'), '2026-08-03T09:00:00.000Z');
assert.ok(toUtcIso('2026-08-03T09:00:00Z').endsWith('Z'));
// idempotent on already-UTC (server may already emit Z)
assert.strictEqual(toUtcIso('2026-08-03T09:00:00Z'), '2026-08-03T09:00:00.000Z');

const timed = dtoToSchedulerEvent({
    start: '2026-08-03T15:00:00+06:00',
    end: '2026-08-03T16:00:00+06:00',
    all_day: false,
    timezone: 'Asia/Omsk',
});
assert.strictEqual(timed.start, '2026-08-03T09:00:00.000Z');
assert.strictEqual(timed.end, '2026-08-03T10:00:00.000Z');
assert.strictEqual(timed.timezone, 'Asia/Omsk'); // metadata kept; Z path ignores it for resolve

// all-day must stay date-only (no UTC datetime conversion)
const allDay = dtoToSchedulerEvent({
    start: '2026-08-03',
    end: '2026-08-04',
    all_day: true,
    timezone: 'UTC',
});
assert.strictEqual(allDay.start, '2026-08-03');
assert.strictEqual(allDay.end, '2026-08-04');

console.log('mappers_check: ok');
