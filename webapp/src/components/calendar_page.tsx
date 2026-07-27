import React, {useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState} from 'react';

import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Divider from '@mui/material/Divider';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import Popover from '@mui/material/Popover';
import Snackbar from '@mui/material/Snackbar';
import Stack from '@mui/material/Stack';
import {ThemeProvider} from '@mui/material/styles';
import Typography from '@mui/material/Typography';
import {EventCalendar} from '@mui/x-scheduler/event-calendar';
import type {SchedulerEvent} from '@mui/x-scheduler/models';

import {
    deleteEvent,
    fetchEvents,
    fetchMe,
    getCaldavConnectURL,
    patchEvent,
    respondEvent,
    type CalendarEventDTO,
    type MeResponse,
    type RespondStatus,
} from '../client';
import {usePluginLocale, useT} from '../i18n';
import {
    dtoFromEventElement,
    dtoToSchedulerEvent,
    schedulerEventToAPITimes,
    schedulerEventsEqual,
    yandexCalendarURL,
} from '../mappers';
import {createMattermostMuiTheme} from '../mm_theme';
import {schedulerLocaleFor} from '../scheduler_locale';
import EventModal, {type EventModalMode} from './event_modal';

function usePluginTheme() {
    const [theme, setTheme] = useState(() => createMattermostMuiTheme());
    useEffect(() => {
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

function measureScrollbarSize(): number {
    const el = document.createElement('div');
    el.style.cssText = 'overflow:scroll;position:absolute;visibility:hidden;width:100px;height:100px';
    document.body.appendChild(el);
    const size = el.offsetWidth - el.clientWidth;
    el.remove();
    return size;
}

function isUnansweredInvite(d: CalendarEventDTO): boolean {
    if (d.is_organizer || d.is_cancelled || !d.response_requested) {
        return false;
    }
    const st = (d.response_status || '').trim();
    return !st || st === 'not_answered';
}

const CalendarPage: React.FC = () => {
    const theme = usePluginTheme();
    const t = useT();
    const locale = usePluginLocale();
    const schedulerLocale = schedulerLocaleFor(locale);
    const calendarWrapRef = useRef<HTMLDivElement>(null);
    const dtosRef = useRef<CalendarEventDTO[]>([]);

    useLayoutEffect(() => {
        const size = measureScrollbarSize();
        calendarWrapRef.current?.style.setProperty('--ycal-scrollbar-size', `${size}px`);
    }, []);

    useEffect(() => {
        const style = document.createElement('style');
        style.setAttribute('data-ycal', 'global-header');
        style.textContent = `
            #global-header,
            [class*="GlobalHeaderContainer"] {
                background-color: var(--sidebar-teambar-bg, var(--sidebar-bg)) !important;
            }
            /* Native MUI event dialog is replaced by EventModal — hide flash. */
            .MuiEventCalendar-eventDialog {
                opacity: 0 !important;
                pointer-events: none !important;
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
    const [respondBusyId, setRespondBusyId] = useState<string | null>(null);

    const [modalOpen, setModalOpen] = useState(false);
    const [modalMode, setModalMode] = useState<EventModalMode>('view');
    const [modalEvent, setModalEvent] = useState<CalendarEventDTO | null>(null);
    const [createDefaults, setCreateDefaults] = useState<{start: string; end: string; all_day: boolean} | undefined>();

    useEffect(() => {
        dtosRef.current = dtos;
    }, [dtos]);

    const pendingInvites = useMemo(() => dtos.filter(isUnansweredInvite), [dtos]);

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

    const applyDto = useCallback((dto: CalendarEventDTO) => {
        setDtos((cur) => {
            const next = [...cur.filter((d) => d.id !== dto.id), dto];
            return next;
        });
        setEvents((cur) => {
            const mapped = dtoToSchedulerEvent(dto);
            if (cur.some((x) => String(x.id) === dto.id)) {
                return cur.map((x) => (String(x.id) === dto.id ? mapped : x));
            }
            return [...cur, mapped];
        });
        setModalEvent((cur) => (cur && cur.id === dto.id ? dto : cur));
    }, []);

    const removeDto = useCallback((id: string) => {
        setDtos((cur) => cur.filter((d) => d.id !== id));
        setEvents((cur) => cur.filter((x) => String(x.id) !== id));
        setModalEvent((cur) => (cur && cur.id === id ? null : cur));
    }, []);

    const onRespond = useCallback(async (id: string, status: RespondStatus) => {
        setRespondBusyId(id);
        try {
            const updated = await respondEvent(id, status);
            if (!updated) {
                removeDto(id);
                setModalOpen(false);
                return;
            }
            applyDto(updated);
        } catch (e: any) {
            setToast(e?.message || t('ycal.webapp.error_respond'));
        } finally {
            setRespondBusyId(null);
        }
    }, [applyDto, removeDto, t]);

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
                // Native create disabled — ignore optimistic inserts without modal.
                setEvents((cur) => cur.filter((x) => String(x.id) !== id));
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
                    applyDto(updated);
                } catch (e: any) {
                    setToast(e?.message || t('ycal.webapp.error_update'));
                    setEvents((cur) => cur.map((x) => (String(x.id) === id ? prev : x)));
                }
            })());
        }
        await Promise.all(patches);
    }, [applyDto, dtos, events, t]);

    const openCreate = useCallback((defaults?: {start: string; end: string; all_day: boolean}) => {
        setInviteAnchor(null);
        setCreateDefaults(defaults);
        setModalMode('create');
        setModalEvent(null);
        setModalOpen(true);
    }, []);

    const openEvent = useCallback((dto: CalendarEventDTO) => {
        setInviteAnchor(null);
        setCreateDefaults(undefined);
        setModalMode('view');
        setModalEvent(dto);
        setModalOpen(true);
    }, []);

    const closeModal = useCallback(() => {
        setModalOpen(false);
        setCreateDefaults(undefined);
    }, []);

    useEffect(() => {
        const root = calendarWrapRef.current;
        if (!root) {
            return;
        }
        const onCaptureClick = (ev: MouseEvent) => {
            const target = ev.target as Element | null;
            if (!target) {
                return;
            }
            const chip = target.closest(
                '.MuiEventCalendar-timeGridEvent, .MuiEventCalendar-dayGridEvent, .MuiEventCalendar-eventItem',
            );
            if (!chip || chip.className.includes('Placeholder')) {
                return;
            }
            const dto = dtoFromEventElement(chip, dtosRef.current);
            if (!dto) {
                return;
            }
            ev.preventDefault();
            ev.stopPropagation();
            openEvent(dto);
        };
        root.addEventListener('click', onCaptureClick, true);
        return () => root.removeEventListener('click', onCaptureClick, true);
    }, [openEvent]);

    // Empty-cell create opens native MUI EventDialog; steal times and show EventModal instead.
    useEffect(() => {
        const inputVal = (dialog: Element, name: string) =>
            (dialog.querySelector(`input[name="${name}"]`) as HTMLInputElement | null)?.value?.trim() || '';

        const stealDialog = (dialog: HTMLElement, attempt = 0) => {
            if (dialog.dataset.ycalStolen === '1') {
                return;
            }

            const startDate = inputVal(dialog, 'startDate');
            if (!startDate && attempt < 12) {
                window.setTimeout(() => stealDialog(dialog, attempt + 1), 16);
                return;
            }
            dialog.dataset.ycalStolen = '1';

            const startTime = inputVal(dialog, 'startTime');
            const endDate = inputVal(dialog, 'endDate') || startDate;
            const endTime = inputVal(dialog, 'endTime');
            const allDaySwitch = dialog.querySelector('.MuiSwitch-input') as HTMLInputElement | null;
            const allDay = Boolean(allDaySwitch?.checked) || (!startTime && Boolean(startDate));

            const isCreate = Boolean(
                calendarWrapRef.current?.querySelector(
                    '.MuiEventCalendar-timeGridEventPlaceholder, .MuiEventCalendar-dayGridEventPlaceholder',
                ),
            );

            const closeBtn = dialog.querySelector(
                '.MuiEventCalendar-eventDialogCloseButton',
            ) as HTMLElement | null;
            closeBtn?.click();
            if (!closeBtn) {
                document.dispatchEvent(new KeyboardEvent('keydown', {key: 'Escape', bubbles: true}));
            }

            if (isCreate && startDate) {
                openCreate(
                    allDay
                        ? {start: startDate, end: endDate, all_day: true}
                        : {
                            start: `${startDate}T${(startTime || '09:00').slice(0, 5)}`,
                            end: `${endDate}T${(endTime || '09:30').slice(0, 5)}`,
                            all_day: false,
                        },
                );
                return;
            }

            const editing = calendarWrapRef.current?.querySelector(
                '.MuiEventCalendar-timeGridEvent[data-editing], .MuiEventCalendar-dayGridEvent[data-editing], [data-editing]',
            );
            if (editing) {
                const dto = dtoFromEventElement(editing, dtosRef.current);
                if (dto) {
                    openEvent(dto);
                }
            }
        };

        const scan = () => {
            document.querySelectorAll<HTMLElement>('.MuiEventCalendar-eventDialog').forEach((dialog) => {
                stealDialog(dialog);
            });
        };

        const observer = new MutationObserver(scan);
        observer.observe(document.body, {childList: true, subtree: true});
        scan();
        return () => observer.disconnect();
    }, [openCreate, openEvent]);

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
                    <Button size='small' variant='contained' onClick={() => openCreate()}>{t('ycal.webapp.create')}</Button>
                    {pendingInvites.length > 0 && (
                        <>
                            <Button size='small' color='inherit' onClick={(e) => setInviteAnchor(e.currentTarget)}>
                                {`${t('ycal.webapp.invites')} (${pendingInvites.length})`}
                            </Button>
                            <Popover
                                anchorEl={inviteAnchor}
                                open={Boolean(inviteAnchor)}
                                onClose={() => setInviteAnchor(null)}
                                anchorOrigin={{vertical: 'bottom', horizontal: 'right'}}
                                transformOrigin={{vertical: 'top', horizontal: 'right'}}
                            >
                                <List dense sx={{minWidth: 320, maxWidth: 420, py: 0}}>
                                    {pendingInvites.map((inv, idx) => (
                                        <React.Fragment key={inv.id}>
                                            {idx > 0 && <Divider/>}
                                            <ListItem
                                                alignItems='flex-start'
                                                sx={{flexDirection: 'column', alignItems: 'stretch', gap: 1, py: 1.5}}
                                            >
                                                <ListItemText
                                                    primary={inv.subject || inv.ical_uid}
                                                    primaryTypographyProps={{
                                                        sx: {cursor: 'pointer', fontWeight: 600},
                                                        onClick: () => openEvent(inv),
                                                    }}
                                                    secondary={inv.start}
                                                />
                                                <Stack direction='row' spacing={1} useFlexGap flexWrap='wrap'>
                                                    <Button
                                                        size='small'
                                                        variant='outlined'
                                                        disabled={respondBusyId === inv.id}
                                                        onClick={(e) => {
                                                            e.stopPropagation();
                                                            onRespond(inv.id, 'accepted');
                                                        }}
                                                    >
                                                        {t('ycal.webapp.accept')}
                                                    </Button>
                                                    <Button
                                                        size='small'
                                                        variant='outlined'
                                                        disabled={respondBusyId === inv.id}
                                                        onClick={(e) => {
                                                            e.stopPropagation();
                                                            onRespond(inv.id, 'tentative');
                                                        }}
                                                    >
                                                        {t('ycal.webapp.tentative')}
                                                    </Button>
                                                    <Button
                                                        size='small'
                                                        color='error'
                                                        variant='outlined'
                                                        disabled={respondBusyId === inv.id}
                                                        onClick={(e) => {
                                                            e.stopPropagation();
                                                            onRespond(inv.id, 'declined');
                                                        }}
                                                    >
                                                        {t('ycal.webapp.decline')}
                                                    </Button>
                                                </Stack>
                                            </ListItem>
                                        </React.Fragment>
                                    ))}
                                </List>
                            </Popover>
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
                            '& .MuiEventCalendar-sidePanelCollapse, & .MuiEventCalendar-sidePanel': {
                                display: 'none !important',
                                width: '0 !important',
                                minWidth: '0 !important',
                                overflow: 'hidden !important',
                            },
                            '& .MuiEventCalendar-headerToolbarSidePanelToggle, & .MuiEventCalendar-headerToolbarLeftElement > .MuiIconButton-root:first-of-type, & button[aria-label="Open side panel"], & button[aria-label="Close side panel"]': {
                                display: 'none !important',
                            },
                            '& .MuiEventCalendar-headerToolbar': {
                                px: 2,
                                boxSizing: 'border-box',
                            },
                            '& .MuiEventCalendar-headerToolbarLeftElement': {
                                gap: 0,
                            },
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

                <EventModal
                    open={modalOpen}
                    mode={modalMode}
                    event={modalEvent}
                    meEmail={me?.email}
                    createDefaults={createDefaults}
                    onClose={closeModal}
                    onSaved={(dto) => {
                        applyDto(dto);
                    }}
                    onDeleted={(id) => {
                        removeDto(id);
                    }}
                    onResponded={(id, dto) => {
                        if (!dto) {
                            removeDto(id);
                            closeModal();
                            return;
                        }
                        applyDto(dto);
                    }}
                />

                <Snackbar open={Boolean(toast)} autoHideDuration={5000} onClose={() => setToast(null)} message={toast || ''}/>
            </Box>
        </ThemeProvider>
    );
};

export default CalendarPage;
