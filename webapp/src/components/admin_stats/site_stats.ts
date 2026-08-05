import {AnalyticsVisualizationType, type PluginAnalyticsRow} from '@mattermost/types/admin';

import type {AdminStats} from '../../client';

const AWAY_COLOR = '#ffbc1f';
const DND_COLOR = '#d24b4e';
const NOT_SET_COLOR = '#9899a3';

const emptyStats: AdminStats = {
    connected_users: 0,
    inactive_users: 0,
    subscriptions: 0,
    receive_reminders: 0,
    daily_summary_enabled: 0,
    set_custom_status: 0,
    status_away: 0,
    status_dnd: 0,
    status_not_set: 0,
    with_channel_events: 0,
    with_active_events: 0,
};

function tr(catalog: Record<string, string>, id: string, fallback: string): string {
    return catalog[id] || fallback;
}

/** Maps admin stats API payload to Mattermost Site Statistics panels. */
export function convertAdminStatsToPanels(
    data: AdminStats | null | undefined,
    catalog: Record<string, string> = {},
): Record<string, PluginAnalyticsRow> {
    const stats = {...emptyStats, ...(data || {})};

    return {
        ycal_connected_users: {
            id: 'ycal_connected_users',
            name: tr(catalog, 'ycal.admin.stats.connected', 'Connected'),
            icon: 'fa-users',
            value: stats.connected_users,
            visualizationType: AnalyticsVisualizationType.Count,
        },
        ycal_inactive_users: {
            id: 'ycal_inactive_users',
            name: tr(catalog, 'ycal.admin.stats.inactive', 'Inactive'),
            icon: 'fa-user-times',
            value: stats.inactive_users,
            visualizationType: AnalyticsVisualizationType.Count,
        },
        ycal_reminders: {
            id: 'ycal_reminders',
            name: tr(catalog, 'ycal.admin.stats.reminders', 'Reminders'),
            icon: 'fa-clock-o',
            value: stats.receive_reminders,
            visualizationType: AnalyticsVisualizationType.Count,
        },
        ycal_daily_summary: {
            id: 'ycal_daily_summary',
            name: tr(catalog, 'ycal.admin.stats.daily_summary', 'Daily summary'),
            icon: 'fa-newspaper-o',
            value: stats.daily_summary_enabled,
            visualizationType: AnalyticsVisualizationType.Count,
        },
        ycal_custom_status: {
            id: 'ycal_custom_status',
            name: tr(catalog, 'ycal.admin.stats.custom_status', 'Custom status'),
            icon: 'fa-smile-o',
            value: stats.set_custom_status,
            visualizationType: AnalyticsVisualizationType.Count,
        },
        ycal_in_meeting: {
            id: 'ycal_in_meeting',
            name: tr(catalog, 'ycal.admin.stats.in_meeting', 'In a meeting now'),
            icon: 'fa-calendar',
            value: stats.with_active_events,
            visualizationType: AnalyticsVisualizationType.Count,
        },
        ycal_status_distribution: {
            id: 'ycal_status_distribution',
            name: tr(catalog, 'ycal.admin.stats.group.status', 'Status distribution'),
            visualizationType: AnalyticsVisualizationType.DoughnutChart,
            value: {
                labels: [
                    tr(catalog, 'ycal.admin.stats.status_away', 'Away'),
                    tr(catalog, 'ycal.admin.stats.status_dnd', 'Do not disturb'),
                    tr(catalog, 'ycal.admin.stats.status_not_set', 'Not set'),
                ],
                datasets: [{
                    data: [stats.status_away, stats.status_dnd, stats.status_not_set],
                    backgroundColor: [AWAY_COLOR, DND_COLOR, NOT_SET_COLOR],
                    hoverBackgroundColor: [AWAY_COLOR, DND_COLOR, NOT_SET_COLOR],
                }],
            },
        },
    };
}
