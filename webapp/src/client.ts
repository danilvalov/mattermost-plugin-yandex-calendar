import {Client4} from 'mattermost-redux/client';

import manifest from 'manifest';

export type CalendarEventDTO = {
    id: string;
    ical_uid: string;
    subject: string;
    description?: string;
    location?: string;
    start: string;
    end: string;
    all_day: boolean;
    timezone: string;
    editable: boolean;
    is_organizer: boolean;
    is_recurring: boolean;
    is_cancelled: boolean;
    response_requested: boolean;
    response_status?: string;
    weblink?: string;
    attendees?: Array<{name?: string; email: string; status?: string}>;
};

export type MeResponse = {
    is_connected: boolean;
    email?: string;
    timezone?: string;
};

export function getPluginServerRoute(): string {
    // Prefer Mattermost SiteURL path (subpath installs) via Client4.url.
    let basePath = '';
    try {
        const siteURL = typeof Client4.getUrl === 'function' ? Client4.getUrl() : '';
        if (siteURL) {
            const u = new URL(siteURL, window.location.origin);
            basePath = u.pathname.replace(/\/$/, '');
        } else {
            const basename = (window as any).basename;
            if (typeof basename === 'string' && basename) {
                basePath = basename.replace(/\/$/, '');
            }
        }
    } catch {
        // ignore
    }
    return `${basePath}/plugins/${manifest.id}`;
}

export function getCaldavConnectURL(): string {
    return `${getPluginServerRoute()}/caldav/connect`;
}

async function pluginFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
    const url = `${getPluginServerRoute()}${path}`;
    const opts = Client4.getOptions(options);
    const res = await fetch(url, opts);
    if (res.status === 204) {
        return undefined as T;
    }
    const text = await res.text();
    let body: any = null;
    if (text) {
        try {
            body = JSON.parse(text);
        } catch {
            body = text;
        }
    }
    if (!res.ok) {
        const msg = body?.error || body?.details || text || res.statusText;
        throw new Error(typeof msg === 'string' ? msg : 'request failed');
    }
    return body as T;
}

export function fetchMe(): Promise<MeResponse> {
    return pluginFetch('/api/v1/me');
}

export function fetchEvents(from: string, to: string): Promise<CalendarEventDTO[]> {
    const q = new URLSearchParams({from, to});
    return pluginFetch(`/api/v1/events?${q.toString()}`);
}

export function fetchEvent(id: string): Promise<CalendarEventDTO> {
    const q = new URLSearchParams({id});
    return pluginFetch(`/api/v1/events/get?${q.toString()}`);
}

export function createEvent(payload: {
    subject: string;
    start: string;
    end: string;
    all_day: boolean;
    description?: string;
    location?: string;
}): Promise<CalendarEventDTO> {
    return pluginFetch('/api/v1/events/create', {
        method: 'POST',
        body: JSON.stringify(payload),
    });
}

export function patchEvent(payload: {
    id: string;
    subject?: string;
    start?: string;
    end?: string;
    all_day?: boolean;
    description?: string;
    location?: string;
}): Promise<CalendarEventDTO> {
    return pluginFetch('/api/v1/events', {
        method: 'PATCH',
        body: JSON.stringify(payload),
    });
}

export function deleteEvent(id: string): Promise<void> {
    const q = new URLSearchParams({id});
    return pluginFetch(`/api/v1/events?${q.toString()}`, {method: 'DELETE'});
}

export function respondEvent(id: string, status: 'accepted' | 'declined' | 'tentative'): Promise<CalendarEventDTO> {
    return pluginFetch('/api/v1/events/respond', {
        method: 'POST',
        body: JSON.stringify({id, status}),
    });
}
