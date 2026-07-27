import React, {useCallback, useEffect, useMemo, useState} from 'react';

import Alert from '@mui/material/Alert';
import Autocomplete from '@mui/material/Autocomplete';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogTitle from '@mui/material/DialogTitle';
import FormControlLabel from '@mui/material/FormControlLabel';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import Box from '@mui/material/Box';

import {
    createEvent as apiCreateEvent,
    deleteEvent,
    patchEvent,
    respondEvent,
    searchMMUsers,
    type CalendarEventDTO,
    type MMUserHit,
    type RespondStatus,
} from '../client';
import {exclusiveToInclusiveDate, inclusiveToExclusiveDate, toDateOnly} from '../dates';
import {useT} from '../i18n';
import {linkifyTextNodes} from '../linkify';
import {yandexCalendarURL} from '../mappers';

export type EventModalMode = 'create' | 'view';

type AttendeeDraft = {email: string; name?: string; status?: string};

type Props = {
    open: boolean;
    mode: EventModalMode;
    event: CalendarEventDTO | null;
    meEmail?: string;
    createDefaults?: {start: string; end: string; all_day: boolean};
    onClose: () => void;
    onSaved: (dto: CalendarEventDTO) => void;
    onDeleted: (id: string) => void;
    onResponded: (id: string, dto: CalendarEventDTO | null) => void;
};

function pad(n: number) {
    return String(n).padStart(2, '0');
}

function toDateTimeLocalValue(iso: string): string {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) {
        return '';
    }
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function fromDateTimeLocalValue(v: string): string {
    // Wall-clock local ISO without offset — server parses in user TZ.
    if (v.length === 16) {
        return `${v}:00`;
    }
    return v;
}

function todayDateOnly(): string {
    const d = new Date();
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/** YYYY-MM-DD from date or datetime-local value; empty if unusable. */
function datePart(v: string): string {
    if (/^\d{4}-\d{2}-\d{2}/.test(v)) {
        return v.slice(0, 10);
    }
    return '';
}

function defaultTimedRangeOnDates(startDate: string, endDate: string): {start: string; end: string} {
    const base = new Date();
    base.setMinutes(0, 0, 0);
    base.setHours(base.getHours() + 1);
    const start = `${startDate}T${pad(base.getHours())}:${pad(base.getMinutes())}`;
    const endBase = new Date(base);
    endBase.setHours(endBase.getHours() + 1);
    if (endDate > startDate) {
        return {start, end: `${endDate}T${pad(base.getHours())}:${pad(base.getMinutes())}`};
    }
    return {start, end: `${startDate}T${pad(endBase.getHours())}:${pad(endBase.getMinutes())}`};
}

function defaultCreateTimes(): {start: string; end: string; all_day: boolean} {
    const start = new Date();
    start.setMinutes(0, 0, 0);
    start.setHours(start.getHours() + 1);
    const end = new Date(start);
    end.setHours(end.getHours() + 1);
    return {
        start: toDateTimeLocalValue(start.toISOString()),
        end: toDateTimeLocalValue(end.toISOString()),
        all_day: false,
    };
}

function statusLabel(t: (id: string, fb?: string) => string, status?: string): string {
    switch (status) {
    case 'accepted':
        return t('ycal.webapp.response_accepted');
    case 'declined':
        return t('ycal.webapp.response_declined');
    case 'tentative':
        return t('ycal.webapp.response_tentative');
    default:
        return t('ycal.webapp.response_none');
    }
}

function isValidEmail(raw: string): boolean {
    const e = raw.trim().toLowerCase();
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(e);
}

const EventModal: React.FC<Props> = ({
    open,
    mode,
    event,
    meEmail,
    createDefaults,
    onClose,
    onSaved,
    onDeleted,
    onResponded,
}) => {
    const t = useT();
    const editable = mode === 'create' || Boolean(event?.editable);
    const isInvitee = Boolean(event && !event.is_organizer && !event.is_cancelled && event.response_requested);
    const recurring = Boolean(event?.is_recurring);

    const [subject, setSubject] = useState('');
    const [allDay, setAllDay] = useState(false);
    const [start, setStart] = useState('');
    const [end, setEnd] = useState('');
    const [location, setLocation] = useState('');
    const [description, setDescription] = useState('');
    const [attendees, setAttendees] = useState<AttendeeDraft[]>([]);
    const [connectTelemost, setConnectTelemost] = useState(false);
    const [searchInput, setSearchInput] = useState('');
    const [options, setOptions] = useState<MMUserHit[]>([]);
    const [searching, setSearching] = useState(false);
    const [busy, setBusy] = useState(false);
    const [respondBusy, setRespondBusy] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        if (!open) {
            return;
        }
        setError(null);
        if (mode === 'create') {
            const d = createDefaults || defaultCreateTimes();
            setSubject('');
            setAllDay(Boolean(d.all_day));
            setStart(d.all_day ? toDateOnly(d.start) : (d.start.includes('T') ? d.start.slice(0, 16) : toDateTimeLocalValue(d.start)));
            setEnd(d.all_day ? toDateOnly(d.end) : (d.end.includes('T') ? d.end.slice(0, 16) : toDateTimeLocalValue(d.end)));
            setLocation('');
            setDescription('');
            setAttendees([]);
            setConnectTelemost(false);
            return;
        }
        if (!event) {
            return;
        }
        setSubject(event.subject || '');
        setAllDay(event.all_day);
        if (event.all_day) {
            setStart(toDateOnly(event.start));
            setEnd(exclusiveToInclusiveDate(event.end));
        } else {
            setStart(toDateTimeLocalValue(event.start));
            setEnd(toDateTimeLocalValue(event.end));
        }
        setLocation(event.location || '');
        setDescription(event.description || '');
        setAttendees((event.attendees || []).map((a) => ({
            email: a.email,
            name: a.name,
            status: a.status,
        })));
        setConnectTelemost(false);
    }, [open, mode, event, createDefaults]);

    useEffect(() => {
        if (!editable || searchInput.trim().length < 2) {
            setOptions([]);
            return;
        }
        let cancelled = false;
        const handle = window.setTimeout(async () => {
            setSearching(true);
            try {
                const hits = await searchMMUsers(searchInput.trim());
                if (!cancelled) {
                    setOptions(hits);
                }
            } catch {
                if (!cancelled) {
                    setOptions([]);
                }
            } finally {
                if (!cancelled) {
                    setSearching(false);
                }
            }
        }, 300);
        return () => {
            cancelled = true;
            window.clearTimeout(handle);
        };
    }, [searchInput, editable]);

    const meNorm = (meEmail || '').trim().toLowerCase();
    const selectedEmails = useMemo(() => new Set(attendees.map((a) => a.email.toLowerCase())), [attendees]);
    const filteredOptions = useMemo(
        () => options.filter((o) => {
            const e = o.email.toLowerCase();
            return e && e !== meNorm && !selectedEmails.has(e);
        }),
        [options, meNorm, selectedEmails],
    );

    const tryAddAttendee = useCallback((emailRaw: string, name?: string) => {
        const email = emailRaw.trim().toLowerCase();
        if (!isValidEmail(email) || email === meNorm) {
            return false;
        }
        let added = false;
        setAttendees((cur) => {
            if (cur.some((a) => a.email.toLowerCase() === email)) {
                return cur;
            }
            added = true;
            return [...cur, {email, name}];
        });
        setSearchInput('');
        setOptions([]);
        return added;
    }, [meNorm]);

    const canEditFields = editable && !recurring;

    const onSave = useCallback(async () => {
        if (!canEditFields && mode !== 'create') {
            return;
        }
        setBusy(true);
        setError(null);
        try {
            const emails = attendees.map((a) => a.email);
            if (mode === 'create') {
                const payload = allDay
                    ? {
                        subject: subject.trim() || t('ycal.webapp.new_event'),
                        start: toDateOnly(start),
                        end: inclusiveToExclusiveDate(toDateOnly(end)),
                        all_day: true,
                        description: description || undefined,
                        location: location || undefined,
                        attendees: emails,
                        ...(connectTelemost ? {telemost: true} : {}),
                    }
                    : {
                        subject: subject.trim() || t('ycal.webapp.new_event'),
                        start: fromDateTimeLocalValue(start),
                        end: fromDateTimeLocalValue(end),
                        all_day: false,
                        description: description || undefined,
                        location: location || undefined,
                        attendees: emails,
                        ...(connectTelemost ? {telemost: true} : {}),
                    };
                const created = await apiCreateEvent(payload);
                onSaved(created);
                onClose();
                return;
            }
            if (!event) {
                return;
            }
            // Omit attendees unless the set changed — avoids RewriteAttendees on every title/time edit.
            const origEmails = (event.attendees || []).map((a) => a.email.toLowerCase()).sort();
            const nextEmails = [...emails].map((e) => e.toLowerCase()).sort();
            const attendeesChanged = origEmails.length !== nextEmails.length ||
                origEmails.some((e, i) => e !== nextEmails[i]);
            const wantTelemost = connectTelemost && !event.conference_url;
            const updated = await patchEvent({
                id: event.id,
                subject: subject.trim() || event.subject,
                all_day: allDay,
                start: allDay ? toDateOnly(start) : fromDateTimeLocalValue(start),
                end: allDay ? inclusiveToExclusiveDate(toDateOnly(end)) : fromDateTimeLocalValue(end),
                description,
                location,
                ...(attendeesChanged ? {attendees: emails} : {}),
                ...(wantTelemost ? {telemost: true} : {}),
            });
            onSaved(updated);
            onClose();
        } catch (e: any) {
            setError(e?.message || t('ycal.webapp.error_update'));
        } finally {
            setBusy(false);
        }
    }, [allDay, attendees, canEditFields, connectTelemost, description, end, event, location, mode, onClose, onSaved, start, subject, t]);

    const onDelete = useCallback(async () => {
        if (!event?.editable) {
            return;
        }
        if (!window.confirm(t('ycal.webapp.delete_confirm'))) {
            return;
        }
        setBusy(true);
        setError(null);
        try {
            await deleteEvent(event.id);
            onDeleted(event.id);
            onClose();
        } catch (e: any) {
            setError(e?.message || t('ycal.webapp.error_delete'));
        } finally {
            setBusy(false);
        }
    }, [event, onClose, onDeleted, t]);

    const onRespond = useCallback(async (status: RespondStatus) => {
        if (!event) {
            return;
        }
        setRespondBusy(true);
        setError(null);
        try {
            const updated = await respondEvent(event.id, status);
            onResponded(event.id, updated);
            if (!updated) {
                onClose();
            }
        } catch (e: any) {
            setError(e?.message || t('ycal.webapp.error_respond'));
        } finally {
            setRespondBusy(false);
        }
    }, [event, onClose, onResponded, t]);

    const title = mode === 'create' ? t('ycal.webapp.new_event') : (event?.subject || t('ycal.webapp.product_name'));

    return (
        <Dialog open={open} onClose={onClose} fullWidth maxWidth='sm'>
            <DialogTitle>{title}</DialogTitle>
            <DialogContent dividers>
                <Stack spacing={2}>
                    {error && <Alert severity='error'>{error}</Alert>}
                    {recurring && (
                        <Alert severity='info'>{t('ycal.webapp.recurring_readonly')}</Alert>
                    )}
                    <TextField
                        label={t('ycal.webapp.field_title')}
                        value={subject}
                        onChange={(e) => setSubject(e.target.value)}
                        disabled={!canEditFields}
                        fullWidth
                    />
                    <FormControlLabel
                        control={
                            <Switch
                                checked={allDay}
                                onChange={(_, v) => {
                                    setAllDay(v);
                                    if (v) {
                                        const s = datePart(start) || todayDateOnly();
                                        let e = datePart(end) || s;
                                        if (e < s) {
                                            e = s;
                                        }
                                        setStart(s);
                                        setEnd(e);
                                        return;
                                    }
                                    const s = datePart(start) || todayDateOnly();
                                    const e = datePart(end) || s;
                                    const timed = defaultTimedRangeOnDates(s, e < s ? s : e);
                                    setStart(timed.start);
                                    setEnd(timed.end);
                                }}
                                disabled={!canEditFields}
                            />
                        }
                        label={t('ycal.webapp.field_all_day')}
                    />
                    <Stack direction={{xs: 'column', sm: 'row'}} spacing={2}>
                        <TextField
                            label={t('ycal.webapp.field_start')}
                            type={allDay ? 'date' : 'datetime-local'}
                            value={start}
                            onChange={(e) => setStart(e.target.value)}
                            disabled={!canEditFields}
                            fullWidth
                            InputLabelProps={{shrink: true}}
                        />
                        <TextField
                            label={allDay ? t('ycal.webapp.field_end_inclusive') : t('ycal.webapp.field_end')}
                            type={allDay ? 'date' : 'datetime-local'}
                            value={end}
                            onChange={(e) => setEnd(e.target.value)}
                            disabled={!canEditFields}
                            fullWidth
                            InputLabelProps={{shrink: true}}
                        />
                    </Stack>
                    <TextField
                        label={t('ycal.webapp.field_location')}
                        value={location}
                        onChange={(e) => setLocation(e.target.value)}
                        disabled={!canEditFields}
                        fullWidth
                    />
                    {(canEditFields || Boolean(event?.conference_url)) && (
                        <Box>
                            <Typography
                                variant='caption'
                                color='text.secondary'
                                sx={{display: 'block', mb: 0.75, ml: 0.25}}
                            >
                                {t('ycal.webapp.field_telemost')}
                            </Typography>
                            {event?.conference_url ? (
                                <Link
                                    href={event.conference_url}
                                    target='_blank'
                                    rel='noopener noreferrer'
                                    underline='hover'
                                    sx={{wordBreak: 'break-all'}}
                                >
                                    {event.conference_url}
                                </Link>
                            ) : (
                                <Stack direction='row' alignItems='center' spacing={1} sx={{minHeight: 32}}>
                                    <Switch
                                        checked={connectTelemost}
                                        onChange={(_, v) => setConnectTelemost(v)}
                                        disabled={!canEditFields}
                                        size='small'
                                        edge='start'
                                        inputProps={{'aria-label': t('ycal.webapp.field_telemost_connect')}}
                                    />
                                    <Typography
                                        component='label'
                                        variant='body1'
                                        onClick={() => canEditFields && setConnectTelemost((v) => !v)}
                                        sx={{
                                            cursor: canEditFields ? 'pointer' : 'default',
                                            userSelect: 'none',
                                            lineHeight: 1.25,
                                        }}
                                    >
                                        {t('ycal.webapp.field_telemost_connect')}
                                    </Typography>
                                </Stack>
                            )}
                        </Box>
                    )}
                    {canEditFields ? (
                        <TextField
                            label={t('ycal.webapp.field_description')}
                            value={description}
                            onChange={(e) => setDescription(e.target.value)}
                            fullWidth
                            multiline
                            minRows={2}
                        />
                    ) : (
                        <Box>
                            <Typography
                                variant='caption'
                                color='text.secondary'
                                sx={{display: 'block', mb: 0.75, ml: 0.25}}
                            >
                                {t('ycal.webapp.field_description')}
                            </Typography>
                            <Box
                                sx={(theme) => ({
                                    border: '1px solid',
                                    borderColor: theme.palette.mode === 'dark' ? 'rgba(255,255,255,0.23)' : 'rgba(0,0,0,0.23)',
                                    borderRadius: `${theme.shape.borderRadius}px`,
                                    px: '14px',
                                    py: '16.5px',
                                    minHeight: 56,
                                    whiteSpace: 'pre-wrap',
                                    wordBreak: 'break-word',
                                    typography: 'body1',
                                    color: 'text.primary',
                                })}
                            >
                                {description.trim()
                                    ? linkifyTextNodes(description)
                                    : (
                                        <Typography component='span' color='text.secondary'>
                                            —
                                        </Typography>
                                    )}
                            </Box>
                        </Box>
                    )}

                    <Typography
                        variant='caption'
                        color='text.secondary'
                        sx={{display: 'block', mb: 0.75, ml: 0.25}}
                    >
                        {t('ycal.webapp.attendees')}
                    </Typography>
                    <Stack direction='row' spacing={1} useFlexGap flexWrap='wrap'>
                        {attendees.length === 0 && (
                            <Typography variant='body2' color='text.secondary'>{t('ycal.webapp.attendees_empty')}</Typography>
                        )}
                        {attendees.map((a) => (
                            <Chip
                                key={a.email}
                                label={`${a.name || a.email}${a.status ? ` · ${statusLabel(t, a.status)}` : ''}`}
                                onDelete={canEditFields ? () => setAttendees((cur) => cur.filter((x) => x.email !== a.email)) : undefined}
                                size='small'
                            />
                        ))}
                    </Stack>
                    {canEditFields && (
                        <Autocomplete
                            freeSolo
                            clearOnBlur
                            selectOnFocus
                            handleHomeEndKeys
                            options={filteredOptions}
                            loading={searching}
                            inputValue={searchInput}
                            onInputChange={(_, v, reason) => {
                                // After select MUI fires reason=reset with the option label — keep field empty.
                                if (reason === 'reset' || reason === 'clear') {
                                    setSearchInput('');
                                    return;
                                }
                                setSearchInput(v);
                            }}
                            getOptionLabel={(o) => (typeof o === 'string' ? o : `${o.display_name} (${o.email})`)}
                            filterOptions={(opts, state) => {
                                const input = state.inputValue.trim();
                                const email = input.toLowerCase();
                                if (
                                    isValidEmail(email) &&
                                    email !== meNorm &&
                                    !selectedEmails.has(email) &&
                                    !opts.some((o) => o.email.toLowerCase() === email)
                                ) {
                                    return [
                                        ...opts,
                                        {id: '', username: '', display_name: input, email} as MMUserHit,
                                    ];
                                }
                                return opts;
                            }}
                            onChange={(_, value) => {
                                if (value == null) {
                                    return;
                                }
                                if (typeof value === 'string') {
                                    tryAddAttendee(value);
                                    return;
                                }
                                tryAddAttendee(value.email, value.display_name);
                            }}
                            renderOption={(props, option) => {
                                const isExternal = !option.id;
                                return (
                                    <li {...props} key={option.id || `email:${option.email}`}>
                                        {isExternal
                                            ? `${t('ycal.webapp.attendees_add_email')}: ${option.email}`
                                            : `${option.display_name} (${option.email})`}
                                    </li>
                                );
                            }}
                            renderInput={(params) => (
                                <TextField
                                    {...params}
                                    label={t('ycal.webapp.attendees_search')}
                                    helperText={t('ycal.webapp.attendees_search_hint')}
                                    InputProps={{
                                        ...params.InputProps,
                                        endAdornment: (
                                            <>
                                                {searching ? <CircularProgress color='inherit' size={16}/> : null}
                                                {params.InputProps.endAdornment}
                                            </>
                                        ),
                                    }}
                                />
                            )}
                        />
                    )}

                    {isInvitee && (
                        <Stack spacing={1}>
                            <Typography variant='subtitle2'>{t('ycal.webapp.response_status')}</Typography>
                            <Typography variant='body2' color='text.secondary'>
                                {statusLabel(t, event?.response_status)}
                            </Typography>
                            <Stack direction='row' spacing={1} useFlexGap flexWrap='wrap'>
                                <Button
                                    size='small'
                                    variant={event?.response_status === 'accepted' ? 'contained' : 'outlined'}
                                    disabled={respondBusy}
                                    onClick={() => onRespond('accepted')}
                                >
                                    {t('ycal.webapp.accept')}
                                </Button>
                                <Button
                                    size='small'
                                    variant={event?.response_status === 'tentative' ? 'contained' : 'outlined'}
                                    disabled={respondBusy}
                                    onClick={() => onRespond('tentative')}
                                >
                                    {t('ycal.webapp.tentative')}
                                </Button>
                                <Button
                                    size='small'
                                    color='error'
                                    variant={event?.response_status === 'declined' ? 'contained' : 'outlined'}
                                    disabled={respondBusy}
                                    onClick={() => onRespond('declined')}
                                >
                                    {t('ycal.webapp.decline')}
                                </Button>
                            </Stack>
                        </Stack>
                    )}
                </Stack>
            </DialogContent>
            <DialogActions sx={{px: 3, py: 2, justifyContent: mode === 'create' ? 'flex-end' : 'space-between'}}>
                {mode !== 'create' && (
                    <Button
                        href={yandexCalendarURL(event)}
                        target='_blank'
                        rel='noopener noreferrer'
                        color='inherit'
                    >
                        {t('ycal.webapp.open_yandex')}
                    </Button>
                )}
                <Stack direction='row' spacing={1}>
                    {event?.editable && mode !== 'create' && (
                        <Button color='error' disabled={busy} onClick={onDelete}>{t('ycal.webapp.delete')}</Button>
                    )}
                    <Button onClick={onClose} disabled={busy}>{t('ycal.webapp.close')}</Button>
                    {canEditFields && (
                        <Button variant='contained' disabled={busy} onClick={onSave}>
                            {busy ? <CircularProgress size={18}/> : t('ycal.webapp.save')}
                        </Button>
                    )}
                </Stack>
            </DialogActions>
        </Dialog>
    );
};

export default EventModal;
