// ponytail: one runnable check for all-day exclusive↔inclusive + MUI/API bridge.
const assert = require('assert');

function exclusiveToInclusiveDate(endExclusive) {
    const d = new Date(toDateOnly(endExclusive) + 'T00:00:00Z');
    d.setUTCDate(d.getUTCDate() - 1);
    return d.toISOString().slice(0, 10);
}

function inclusiveToExclusiveDate(endInclusive) {
    const d = new Date(toDateOnly(endInclusive) + 'T00:00:00Z');
    d.setUTCDate(d.getUTCDate() + 1);
    return d.toISOString().slice(0, 10);
}

function toDateOnly(value) {
    if (value && typeof value === 'object' && value.year != null) {
        const pad = (n) => String(n).padStart(2, '0');
        return `${value.year}-${pad(value.month)}-${pad(value.day)}`;
    }
    const s = String(value ?? '');
    if (/^\d{4}-\d{2}-\d{2}/.test(s)) {
        return s.slice(0, 10);
    }
    return new Date(s).toISOString().slice(0, 10);
}

function schedulerAllDayToAPI(start, end) {
    return {start: toDateOnly(start), end: inclusiveToExclusiveDate(toDateOnly(end))};
}

// one-day all-day: API exclusive Jul28 ↔ MUI inclusive Jul27
assert.strictEqual(exclusiveToInclusiveDate('2026-07-28'), '2026-07-27');
assert.strictEqual(inclusiveToExclusiveDate('2026-07-27'), '2026-07-28');

// MUI same-day create would reject without conversion
const created = schedulerAllDayToAPI('2026-07-27', '2026-07-27');
assert.strictEqual(created.start, '2026-07-27');
assert.strictEqual(created.end, '2026-07-28');

// multi-day inclusive Jul27–29 → exclusive end Jul30
const multi = schedulerAllDayToAPI('2026-07-27', '2026-07-29');
assert.strictEqual(multi.end, '2026-07-30');

console.log('dates_check: ok');
