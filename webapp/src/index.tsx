// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import type {Store} from 'redux';

import type {GlobalState} from '@mattermost/types/store';

import manifest from 'manifest';

import CalendarPage from './components/calendar_page';
import {catalogForLocale, localeFromState} from './i18n';
import type {PluginRegistry} from 'types/mattermost-webapp';

import en from '../i18n/en.json';
import ru from '../i18n/ru.json';

function getTranslations(locale: string): {[key: string]: string} {
    switch (locale) {
    case 'ru':
        return ru;
    default:
        return en;
    }
}

export default class Plugin {
    public async initialize(registry: PluginRegistry, store: Store<GlobalState>) {
        registry.registerTranslations(getTranslations);

        const locale = localeFromState(store.getState(), 'en');
        const productName = catalogForLocale(locale)['ycal.webapp.product_name'] || 'Calendar';

        registry.registerProduct(
            '/yandex-calendar',
            'calendar-outline',
            productName,
            '/yandex-calendar',
            CalendarPage,
            () => null,
            () => null,
            true,
            false,
            true,
        );
    }
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

window.registerPlugin(manifest.id, new Plugin());
