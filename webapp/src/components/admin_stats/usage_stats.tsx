import React, {useEffect, useState} from 'react';

import {type AdminStats, fetchAdminStats, fetchMattermostTotalUsersCount} from '../../client';
import {useT} from '../../i18n';

import './usage_stats.scss';

type LoadState =
    | {status: 'loading'}
    | {status: 'error'}
    | {
        status: 'ready';
        stats: AdminStats;
        totalUsers: number | null;
    };

type Card = {
    key: string;
    value: React.ReactNode;
    label: string;
    primary?: boolean;
};

function formatMessage(template: string, values: Record<string, string | number>): string {
    return Object.keys(values).reduce(
        (out, key) => out.replace(new RegExp(`\\{${key}\\}`, 'g'), String(values[key])),
        template,
    );
}

function StatsGrid({cards}: {cards: Card[]}) {
    return (
        <div className='YcalUsageStats__grid'>
            {cards.map((card) => (
                <div
                    key={card.key}
                    className={`YcalUsageStats__card${card.primary ? ' YcalUsageStats__card--primary' : ''}`}
                >
                    <div className='YcalUsageStats__value'>{card.value}</div>
                    <div className='YcalUsageStats__label'>{card.label}</div>
                </div>
            ))}
        </div>
    );
}

/** Compact read-only usage block for System Console plugin settings (bottom). */
const UsageStats: React.FC = () => {
    const t = useT();
    const [state, setState] = useState<LoadState>({status: 'loading'});

    useEffect(() => {
        let cancelled = false;

        (async () => {
            try {
                const [stats, totalUsers] = await Promise.all([
                    fetchAdminStats(),
                    fetchMattermostTotalUsersCount().catch(() => null),
                ]);
                if (!cancelled) {
                    setState({status: 'ready', stats, totalUsers});
                }
            } catch {
                if (!cancelled) {
                    setState({status: 'error'});
                }
            }
        })();

        return () => {
            cancelled = true;
        };
    }, []);

    if (state.status === 'loading') {
        return (
            <div className='YcalUsageStats'>
                <div className='YcalUsageStats__header'>
                    <h4 className='YcalUsageStats__title'>
                        {t('ycal.admin.stats.title', 'Usage')}
                    </h4>
                    <p className='YcalUsageStats__subtitle YcalUsageStats__subtitle--loading'>
                        {t('ycal.admin.stats.loading', 'Loading…')}
                    </p>
                </div>
                <div className='YcalUsageStats__grids'>
                    {[0, 1, 2].map((row) => (
                        <div
                            key={row}
                            className='YcalUsageStats__grid YcalUsageStats__grid--skeleton'
                        >
                            {[0, 1, 2, 3].map((i) => (
                                <div
                                    key={i}
                                    className='YcalUsageStats__card YcalUsageStats__card--skeleton'
                                />
                            ))}
                        </div>
                    ))}
                </div>
            </div>
        );
    }

    if (state.status === 'error') {
        return (
            <div className='YcalUsageStats'>
                <div className='YcalUsageStats__header'>
                    <h4 className='YcalUsageStats__title'>
                        {t('ycal.admin.stats.title', 'Usage')}
                    </h4>
                    <p className='YcalUsageStats__error'>
                        {t('ycal.admin.stats.error', 'Failed to load usage statistics.')}
                    </p>
                </div>
            </div>
        );
    }

    const {stats, totalUsers} = state;
    const hasTotal = totalUsers !== null && totalUsers > 0;
    const adoption = hasTotal ? Math.round((100 * stats.connected_users) / totalUsers) : null;
    const autoStatus = stats.status_away + stats.status_dnd;

    const rowUsers: Card[] = [
        {
            key: 'connected',
            value: stats.connected_users,
            label: t('ycal.admin.stats.connected', 'Connected'),
            primary: true,
        },
        {
            key: 'adoption',
            value: adoption === null ? '—' : `${adoption}%`,
            label: t('ycal.admin.stats.adoption', 'Adoption'),
        },
        {
            key: 'inactive',
            value: stats.inactive_users,
            label: t('ycal.admin.stats.inactive', 'Inactive'),
        },
        {
            key: 'subscriptions',
            value: stats.subscriptions,
            label: t('ycal.admin.stats.subscriptions', 'Subscriptions'),
        },
    ];

    const rowFeatures: Card[] = [
        {
            key: 'reminders',
            value: stats.receive_reminders,
            label: t('ycal.admin.stats.reminders', 'Reminders'),
        },
        {
            key: 'daily_summary',
            value: stats.daily_summary_enabled,
            label: t('ycal.admin.stats.daily_summary', 'Daily summary'),
        },
        {
            key: 'custom_status',
            value: stats.set_custom_status,
            label: t('ycal.admin.stats.custom_status', 'Custom status'),
        },
        {
            key: 'auto_status',
            value: autoStatus,
            label: t('ycal.admin.stats.auto_status', 'Auto status'),
        },
    ];

    const rowLive: Card[] = [
        {
            key: 'linked',
            value: stats.with_channel_events,
            label: t('ycal.admin.stats.with_channel_events', 'Linked to channel'),
        },
        {
            key: 'in_meeting',
            value: stats.with_active_events,
            label: t('ycal.admin.stats.in_meeting', 'In a meeting now'),
        },
        {
            key: 'status_away',
            value: stats.status_away,
            label: t('ycal.admin.stats.status_away', 'Away'),
        },
        {
            key: 'status_dnd',
            value: stats.status_dnd,
            label: t('ycal.admin.stats.status_dnd', 'Do not disturb'),
        },
    ];

    return (
        <div className='YcalUsageStats'>
            <div className='YcalUsageStats__header'>
                <h4 className='YcalUsageStats__title'>
                    {t('ycal.admin.stats.title', 'Usage')}
                </h4>
                <p className='YcalUsageStats__subtitle'>
                    {t('ycal.admin.stats.subtitle', 'Connected users are counted from stored Yandex Calendar credentials.')}
                </p>
            </div>

            <div className='YcalUsageStats__grids'>
                <StatsGrid cards={rowUsers}/>
                <StatsGrid cards={rowFeatures}/>
                <StatsGrid cards={rowLive}/>
            </div>

            <p className='YcalUsageStats__footer'>
                {hasTotal ? (
                    formatMessage(
                        t('ycal.admin.stats.footer', '{connected} of {total} Mattermost users have connected Yandex Calendar.'),
                        {connected: stats.connected_users, total: totalUsers as number},
                    )
                ) : (
                    formatMessage(
                        t('ycal.admin.stats.footer_no_total', '{connected} Mattermost users have connected Yandex Calendar.'),
                        {connected: stats.connected_users},
                    )
                )}
                {' · '}
                {formatMessage(
                    t('ycal.admin.stats.footer_status', 'Status not set: {count}'),
                    {count: stats.status_not_set},
                )}
            </p>
        </div>
    );
};

export default UsageStats;
