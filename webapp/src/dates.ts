/** Inclusive last day for all-day exclusive end (YYYY-MM-DD). */
export function exclusiveToInclusiveDate(endExclusive: string): string {
    const d = new Date(toDateOnly(endExclusive) + 'T00:00:00Z');
    d.setUTCDate(d.getUTCDate() - 1);
    return d.toISOString().slice(0, 10);
}

export function inclusiveToExclusiveDate(endInclusive: string): string {
    const d = new Date(toDateOnly(endInclusive) + 'T00:00:00Z');
    d.setUTCDate(d.getUTCDate() + 1);
    return d.toISOString().slice(0, 10);
}

/** Normalize Temporal / ISO / date string to YYYY-MM-DD (calendar day). */
export function toDateOnly(value: unknown): string {
    if (value && typeof value === 'object') {
        const v = value as {year?: number; month?: number; day?: number};
        if (typeof v.year === 'number' && typeof v.month === 'number' && typeof v.day === 'number') {
            const pad = (n: number) => String(n).padStart(2, '0');
            return `${v.year}-${pad(v.month)}-${pad(v.day)}`;
        }
    }
    const s = String(value ?? '');
    if (/^\d{4}-\d{2}-\d{2}/.test(s)) {
        return s.slice(0, 10);
    }
    const d = new Date(s);
    if (!Number.isNaN(d.getTime())) {
        // Wall-time / local calendar day for datetime-local-ish values without Z.
        if (!/[zZ]|[+-]\d{2}:?\d{2}$/.test(s)) {
            const pad = (n: number) => String(n).padStart(2, '0');
            return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
        }
        return d.toISOString().slice(0, 10);
    }
    return s.slice(0, 10);
}

/**
 * MUI Scheduler treats all-day end as inclusive (endOfDay); our API uses iCal exclusive end.
 * Convert scheduler start/end → API exclusive date pair.
 */
export function schedulerAllDayToAPI(start: unknown, end: unknown): {start: string; end: string} {
    const startDate = toDateOnly(start);
    const endInclusive = toDateOnly(end);
    return {
        start: startDate,
        end: inclusiveToExclusiveDate(endInclusive),
    };
}
