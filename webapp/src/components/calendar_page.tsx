import React, {useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState} from 'react';

import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import {ThemeProvider} from '@mui/material/styles';
import Typography from '@mui/material/Typography';
import {EventCalendar} from '@mui/x-scheduler/event-calendar';
import type {SchedulerEvent} from '@mui/x-scheduler/models';

import {
    createEvent as apiCreateEvent,
    deleteEvent,
    fetchEvents,
    fetchMe,
    getCaldavConnectURL,
    patchEvent,
    type CalendarEventDTO,
    type MeResponse,
} from '../client';
import {usePluginLocale, useT} from '../i18n';
import {dtoToSchedulerEvent, eventClassForId, schedulerEventToAPITimes, schedulerEventsEqual, yandexCalendarURL} from '../mappers';
import {createMattermostMuiTheme} from '../mm_theme';
import {schedulerLocaleFor} from '../scheduler_locale';

function usePluginTheme() {
    const [theme, setTheme] = useState(() => createMattermostMuiTheme());
    useEffect(() => {
        // Re-read MM CSS vars after product mount (body vars can lag on first paint).
        setTheme(createMattermostMuiTheme());
    }, []);
    return theme;
}

function temporalToDate(value: unknown): Date {
    if (value instanceof Date) {
        return value;
    }
    if (typeof value === 'string' || typeof value === 'number') {
        const d = new Date(value);
        if (!Number.isNaN(d.getTime())) {
            return d;
        }
    }
    const v = value as {year?: number; month?: number; day?: number; epochMilliseconds?: number; toInstant?: () => {epochMilliseconds: number}};
    if (typeof v?.toInstant === 'function') {
        return new Date(v.toInstant().epochMilliseconds);
    }
    if (typeof v?.epochMilliseconds === 'number') {
        return new Date(v.epochMilliseconds);
    }
    if (v?.year != null && v.month != null && v.day != null) {
        return new Date(v.year, v.month - 1, v.day);
    }
    return new Date();
}

function rangeAround(center: Date): {from: Date; to: Date} {
    const from = new Date(center);
    from.setDate(from.getDate() - 14);
    from.setHours(0, 0, 0, 0);
    const to = new Date(center);
    to.setDate(to.getDate() + 28);
    to.setHours(23, 59, 59, 999);
    return {from, to};
}

/** Classic scrollbar width (overlay scrollbars → 0). Used to un-collapse MUI header placeholder. */
function measureScrollbarSize(): number {
    const el = document.createElement('div');
    el.style.cssText = 'overflow:scroll;position:absolute;visibility:hidden;width:100px;height:100px';
    document.body.appendChild(el);
    const size = el.offsetWidth - el.clientWidth;
    el.remove();
    return size;
}

/** Open MUI EventDialog create flow (same as clicking an empty time slot). */
function triggerNativeCreate(root: HTMLElement | null) {
    if (!root) {
        return;
    }
    const layers = [...root.querySelectorAll('.MuiEventCalendar-dayTimeGridColumnInteractiveLayer')] as HTMLElement[];
    const empty = layers.find((l) => !l.querySelector('.MuiEventCalendar-timeGridEvent')) || layers[Math.min(3, layers.length - 1)] || layers[0];
    if (!empty) {
        return;
    }
    const r = empty.getBoundingClientRect();
    const x = r.left + Math.min(24, r.width / 2);
    const y = r.top + 96;
    empty.dispatchEvent(new MouseEvent('click', {
        bubbles: true,
        cancelable: true,
        view: window,
        clientX: x,
        clientY: y,
    }));
}

/** Open MUI EventDialog for an existing event (same as clicking the event chip). */
function triggerNativeEventOpen(root: HTMLElement | null, eventId: string) {
    if (!root) {
        return;
    }
    const token = eventClassForId(eventId);
    const el = root.querySelector(`.${token}`) as HTMLElement | null;
    const clickable = (el?.closest('button, [role="button"]') || el) as HTMLElement | null;
    clickable?.click();
}

const CalendarPage: React.FC = () => {
    const theme = usePluginTheme();
    const t = useT();
    const locale = usePluginLocale();
    const schedulerLocale = schedulerLocaleFor(locale);
    const calendarWrapRef = useRef<HTMLDivElement>(null);

    useLayoutEffect(() => {
        const size = measureScrollbarSize();
        calendarWrapRef.current?.style.setProperty('--ycal-scrollbar-size', `${size}px`);
    }, []);

    // MM GlobalHeader has no background (transparent) + light sidebar-text.
    // On product pages the area behind it is center-channel (white) → header looks "missing".
    useEffect(() => {
        const style = document.createElement('style');
        style.setAttribute('data-ycal', 'global-header');
        style.textContent = `
            #global-header,
            [class*="GlobalHeaderContainer"] {
                background-color: var(--sidebar-teambar-bg, var(--sidebar-bg)) !important;
            }
        `;
        document.head.appendChild(style);
        return () => {
            style.remove();
        };
    }, []);

    const [me, setMe] = useState<MeResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [toast, setToast] = useState<string | null>(null);
    const [dtos, setDtos] = useState<CalendarEventDTO[]>([]);
    const [events, setEvents] = useState<SchedulerEvent[]>([]);
    const [range, setRange] = useState(() => rangeAround(new Date()));
    const [inviteAnchor, setInviteAnchor] = useState<null | HTMLElement>(null);

    const pendingInvites = useMemo(
        () => dtos.filter((d) => !d.is_organizer && d.response_requested && !d.is_cancelled),
        [dtos],
    );

    const loadMe = useCallback(async () => {
        try {
            const m = await fetchMe();
            setMe(m);
            return m;
        } catch (e: any) {
            setMe({is_connected: false});
            setError(e?.message || t('ycal.webapp.error_connection'));
            return null;
        }
    }, [t]);

    const loadEvents = useCallback(async (from: Date, to: Date) => {
        setLoading(true);
        setError(null);
        try {
            const list = await fetchEvents(from.toISOString(), to.toISOString());
            setDtos(list);
            setEvents(list.map(dtoToSchedulerEvent));
        } catch (e: any) {
            setError(e?.message || t('ycal.webapp.error_load'));
        } finally {
            setLoading(false);
        }
    }, [t]);

    useEffect(() => {
        (async () => {
            const m = await loadMe();
            if (m?.is_connected) {
                await loadEvents(range.from, range.to);
            } else {
                setLoading(false);
            }
        })();
    }, [loadMe, loadEvents, range.from, range.to]);

    const refresh = useCallback(() => {
        loadEvents(range.from, range.to);
    }, [loadEvents, range.from, range.to]);

    const onVisibleDateChange = useCallback((visibleDate: unknown) => {
        setRange(rangeAround(temporalToDate(visibleDate)));
    }, []);

    const onEventsChange = useCallback(async (next: SchedulerEvent[]) => {
        const prevById = new Map(events.map((e) => [String(e.id), e]));
        const nextById = new Map(next.map((e) => [String(e.id), e]));
        setEvents(next);

        for (const [id, prev] of prevById) {
            if (nextById.has(id)) {
                continue;
            }
            const dto = dtos.find((d) => d.id === id);
            if (!dto?.editable) {
                continue;
            }
            try {
                await deleteEvent(id);
                setDtos((cur) => cur.filter((d) => d.id !== id));
            } catch (e: any) {
                setToast(e?.message || t('ycal.webapp.error_delete'));
                setEvents((cur) => [...cur, prev]);
            }
        }

        const patches: Promise<void>[] = [];
        for (const ev of next) {
            const id = String(ev.id);
            const prev = prevById.get(id);
            if (!prev) {
                patches.push((async () => {
                    try {
                        const times = schedulerEventToAPITimes(ev);
                        const created = await apiCreateEvent({
                            subject: String(ev.title || 'Event'),
                            ...times,
                            description: ev.description ? String(ev.description) : undefined,
                        });
                        setDtos((cur) => [...cur.filter((d) => d.id !== id), created]);
                        setEvents((cur) => cur.map((x) => (String(x.id) === id ? dtoToSchedulerEvent(created) : x)));
                    } catch (e: any) {
                        setToast(e?.message || t('ycal.webapp.error_create'));
                        setEvents((cur) => cur.filter((x) => String(x.id) !== id));
                    }
                })());
                continue;
            }
            if (schedulerEventsEqual(prev, ev)) {
                continue;
            }
            const dto = dtos.find((d) => d.id === id);
            if (!dto?.editable) {
                continue;
            }
            patches.push((async () => {
                try {
                    const times = schedulerEventToAPITimes(ev);
                    const updated = await patchEvent({
                        id,
                        subject: String(ev.title || dto.subject),
                        ...times,
                        description: ev.description != null ? String(ev.description) : undefined,
                    });
                    setDtos((cur) => cur.map((d) => (d.id === updated.id ? updated : d)));
                    setEvents((cur) => cur.map((x) => (String(x.id) === updated.id ? dtoToSchedulerEvent(updated) : x)));
                } catch (e: any) {
                    setToast(e?.message || t('ycal.webapp.error_update'));
                    setEvents((cur) => cur.map((x) => (String(x.id) === id ? prev : x)));
                }
            })());
        }
        await Promise.all(patches);
    }, [dtos, events, t]);

    const openCreate = useCallback(() => {
        triggerNativeCreate(calendarWrapRef.current);
    }, []);

    const openEvent = useCallback((dto: CalendarEventDTO) => {
        setInviteAnchor(null);
        // Defer so the Menu closes before the dialog mounts.
        requestAnimationFrame(() => triggerNativeEventOpen(calendarWrapRef.current, dto.id));
    }, []);

    if (me && !me.is_connected) {
        return (
            <ThemeProvider theme={theme}>
                <Box
                    sx={{
                        p: 4,
                        height: '100%',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        bgcolor: 'background.default',
                        color: 'text.primary',
                    }}
                >
                    <Stack spacing={2} alignItems='center' maxWidth={480}>
                        <Typography variant='h5'>{t('ycal.webapp.connect_title')}</Typography>
                        <Typography align='center' color='text.secondary'>
                            {t('ycal.webapp.connect_body')}
                        </Typography>
                        <Button variant='contained' href={getCaldavConnectURL()} target='_blank' rel='noopener noreferrer'>
                            {t('ycal.webapp.connect_button')}
                        </Button>
                    </Stack>
                </Box>
            </ThemeProvider>
        );
    }

    return (
        <ThemeProvider theme={theme}>
            <Box
                sx={{
                    height: '100%',
                    display: 'flex',
                    flexDirection: 'column',
                    minHeight: 0,
                    bgcolor: 'background.default',
                    color: 'text.primary',
                }}
            >
                <Stack
                    direction='row'
                    spacing={1}
                    alignItems='center'
                    sx={{px: 2, py: 1, borderBottom: 1, borderColor: 'divider', flexShrink: 0}}
                >
                    <Typography variant='h6' sx={{flex: 1, color: 'text.primary'}}>{t('ycal.webapp.product_name')}</Typography>
                    <Button size='small' variant='contained' onClick={openCreate}>{t('ycal.webapp.create')}</Button>
                    {pendingInvites.length > 0 && (
                        <>
                            <Button size='small' color='inherit' onClick={(e) => setInviteAnchor(e.currentTarget)}>
                                {`${t('ycal.webapp.invites')} (${pendingInvites.length})`}
                            </Button>
                            <Menu
                                anchorEl={inviteAnchor}
                                open={Boolean(inviteAnchor)}
                                onClose={() => setInviteAnchor(null)}
                            >
                                {pendingInvites.map((inv) => (
                                    <MenuItem key={inv.id} onClick={() => openEvent(inv)}>
                                        {inv.subject || inv.ical_uid}
                                    </MenuItem>
                                ))}
                            </Menu>
                        </>
                    )}
                    <Button size='small' color='inherit' onClick={refresh}>{t('ycal.webapp.refresh')}</Button>
                    <Button
                        size='small'
                        color='inherit'
                        href={yandexCalendarURL()}
                        target='_blank'
                        rel='noopener noreferrer'
                    >
                        {t('ycal.webapp.open_yandex')}
                    </Button>
                </Stack>

                {error && (
                    <Alert severity='error' onClose={() => setError(null)} sx={{borderRadius: 0}}>
                        {error}
                    </Alert>
                )}

                <Box sx={{flex: 1, minHeight: 0, position: 'relative'}}>
                    {loading && (
                        <Box
                            sx={{
                                position: 'absolute',
                                inset: 0,
                                display: 'grid',
                                placeItems: 'center',
                                zIndex: 2,
                                bgcolor: (th) => th.palette.mode === 'dark' ? 'rgba(0,0,0,0.45)' : 'rgba(255,255,255,0.5)',
                            }}
                        >
                            <CircularProgress/>
                        </Box>
                    )}
                    <Box
                        ref={calendarWrapRef}
                        sx={{
                            height: '100%',
                            width: '100%',
                            '--ycal-scrollbar-size': '0px',
                            '& .MuiEventCalendar-root': {
                                height: '100%',
                                bgcolor: 'background.default',
                                color: 'text.primary',
                            },
                            // v1: hide unused mini-calendar side panel + its burger toggle.
                            '& .MuiEventCalendar-sidePanelCollapse, & .MuiEventCalendar-sidePanel': {
                                display: 'none !important',
                                width: '0 !important',
                                minWidth: '0 !important',
                                overflow: 'hidden !important',
                            },
                            '& .MuiEventCalendar-headerToolbarSidePanelToggle, & .MuiEventCalendar-headerToolbarLeftElement > .MuiIconButton-root:first-of-type, & button[aria-label="Open side panel"], & button[aria-label="Close side panel"]': {
                                display: 'none !important',
                            },
                            // Match product header Stack px:2 so month label lines up with «Календарь».
                            '& .MuiEventCalendar-headerToolbar': {
                                px: 2,
                                boxSizing: 'border-box',
                            },
                            '& .MuiEventCalendar-headerToolbarLeftElement': {
                                gap: 0,
                            },
                            // MUI placeholder collapses to 0 under fit-content grid; force classic scrollbar width.
                            '& .MuiEventCalendar-dayTimeGridScrollablePlaceholder': {
                                width: 'var(--ycal-scrollbar-size) !important',
                                minWidth: 'var(--ycal-scrollbar-size) !important',
                                flexShrink: 0,
                            },
                            '& .MuiEventCalendar-dayTimeGridAllDayEventsCell, & .MuiEventCalendar-dayTimeGridTimeAxis, & .MuiPaper-root': {
                                bgcolor: 'background.paper',
                                color: 'text.primary',
                            },
                            '& .MuiEventCalendar-dayTimeGridTimeAxisText, & .MuiEventCalendar-headerToolbarLabel, & .MuiEventCalendar-dayTimeGridHeaderDayName': {
                                color: 'text.secondary',
                            },
                            '& .MuiEventCalendar-dayTimeGridHeaderDayNumber': {
                                color: 'text.primary',
                            },
                        }}
                    >
                        <EventCalendar
                            events={events}
                            onEventsChange={onEventsChange}
                            onVisibleDateChange={onVisibleDateChange}
                            areEventsDraggable={true}
                            areEventsResizable={true}
                            eventCreation={true}
                            displayTimezone={me?.timezone || 'default'}
                            dateLocale={schedulerLocale.dateLocale}
                            localeText={schedulerLocale.localeText}
                            defaultPreferences={{isSidePanelOpen: false, ampm: false}}
                        />
                    </Box>
                </Box>

                <Snackbar open={Boolean(toast)} autoHideDuration={5000} onClose={() => setToast(null)} message={toast || ''}/>
            </Box>
        </ThemeProvider>
    );
};

export default CalendarPage;
