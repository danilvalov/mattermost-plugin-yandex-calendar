import {useCallback} from 'react';
import {useSelector} from 'react-redux';

import type {GlobalState} from '@mattermost/types/store';

import en from '../i18n/en.json';
import ru from '../i18n/ru.json';

const catalogs: Record<string, Record<string, string>> = {
    en,
    ru,
};

export function localeLang(locale: string): string {
    return (locale || 'en').toLowerCase().slice(0, 2);
}

export function translate(locale: string, id: string, fallback?: string): string {
    const lang = localeLang(locale);
    const catalog = catalogs[lang] || catalogs.en;
    return catalog[id] || catalogs.en[id] || fallback || id;
}

function localeFromState(state: GlobalState, fallback = 'en'): string {
    const id = state.entities?.users?.currentUserId;
    const profile = id ? state.entities.users.profiles?.[id] : undefined;
    return profile?.locale || fallback;
}

export function usePluginLocale(): string {
    return useSelector((state: GlobalState) => localeFromState(state, 'en'));
}

/** Reactive translator bound to Mattermost user locale. */
export function useT(): (id: string, fallback?: string) => string {
    const locale = usePluginLocale();
    return useCallback((id: string, fallback?: string) => translate(locale, id, fallback), [locale]);
}

/**
 * Non-hook fallback. Prefer useT() in components —
 * document.lang often stays "en" while MM user locale is "ru".
 */
export function t(id: string, fallback?: string): string {
    const lang = (document.documentElement.lang || 'en').toLowerCase().slice(0, 2);
    return translate(lang, id, fallback);
}

export function catalogForLocale(locale: string): Record<string, string> {
    return catalogs[localeLang(locale)] || catalogs.en;
}

export {localeFromState};
