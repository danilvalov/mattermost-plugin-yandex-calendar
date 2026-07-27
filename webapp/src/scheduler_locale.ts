import type {Locale as DateFnsLocale} from 'date-fns/locale';
import {enUS as dateFnsEnUS} from 'date-fns/locale/en-US';
import {ru as dateFnsRu} from 'date-fns/locale/ru';

import {localeLang} from './i18n';

/** Nominative, title-case — MUI header uses MMMM (formatting), which is genitive in stock date-fns ru. */
const RU_MONTHS_NOM = [
    'Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь',
    'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь',
] as const;

/** ponytail: wide months always title-style; day-prefixed phrases (PPPP) may read «27 Июль». */
const dateFnsRuTitles: DateFnsLocale = {
    ...dateFnsRu,
    localize: {
        ...dateFnsRu.localize!,
        month: (n, options) => {
            if ((options?.width ?? 'wide') === 'wide') {
                return RU_MONTHS_NOM[n];
            }
            return dateFnsRu.localize!.month(n, options);
        },
    },
};

/** MUI EventCalendar localeText — upstream ruRU pack is empty stubs. */
const localeTextRu = {
    // ViewSwitcher / toolbar
    today: 'Сегодня',
    day: 'День',
    week: 'Неделя',
    month: 'Месяц',
    agenda: 'Повестка',
    other: 'Другое',
    time: 'Время',
    days: 'Дни',
    months: 'Месяцы',
    weeks: 'Недели',
    years: 'Годы',
    allDay: 'Весь день',
    preferencesMenu: 'Настройки',
    showWeekends: 'Показывать выходные',
    showWeekNumber: 'Номер недели',
    showEmptyDaysInAgenda: 'Показывать пустые дни',
    timeFormat: 'Формат времени',
    amPm12h: '12 часов (1:00PM)',
    hour24h: '24 часа (13:00)',
    viewSpecificOptions: (view: string) => `Параметры: ${view}`,
    closeSidePanel: 'Закрыть боковую панель',
    openSidePanel: 'Открыть боковую панель',
    nextTimeSpan: (timeSpan: string) => `Следующ.: ${timeSpan}`,
    previousTimeSpan: (timeSpan: string) => `Предыдущ.: ${timeSpan}`,
    hiddenEvents: (n: number) => `Ещё ${n}…`,
    weekAbbreviation: 'Н',
    weekNumberAriaLabel: (weekNumber: number) => `Неделя ${weekNumber}`,
    eventItemMultiDayLabel: (endDate: string) => `До ${endDate}`,
    miniCalendarLabel: 'Календарь',
    miniCalendarGoToPreviousMonth: 'Предыдущий месяц',
    miniCalendarGoToNextMonth: 'Следующий месяц',
    resourcesLabel: 'Ресурсы',
    resourcesLegendSectionLabel: 'Легенда ресурсов',
    hideEventsLabel: (resourceName: string) => `Скрыть: ${resourceName}`,
    showEventsLabel: (resourceName: string) => `Показать: ${resourceName}`,
    resourceAriaLabel: (resourceName: string) => `Ресурс: ${resourceName}`,
    noResourceAriaLabel: 'Без ресурса',
    timelineResourceTitleHeader: 'Ресурс',
    // EventDialog (FormContent / ReadonlyContent)
    eventTitleAriaLabel: 'Название события',
    dateTimeSectionLabel: 'Дата и время',
    resourceColorSectionLabel: 'Ресурс и цвет',
    allDayLabel: 'Весь день',
    closeButtonAriaLabel: 'Закрыть',
    closeButtonLabel: 'Закрыть',
    deleteEvent: 'Удалить событие',
    descriptionLabel: 'Описание',
    endDateLabel: 'Дата окончания',
    endTimeLabel: 'Время окончания',
    startDateLabel: 'Дата начала',
    startTimeLabel: 'Время начала',
    colorPickerLabel: 'Цвет события',
    generalTabLabel: 'Основное',
    labelNoResource: 'Без ресурса',
    labelInvalidResource: 'Неверный ресурс',
    resourceLabel: 'Ресурс',
    saveChanges: 'Сохранить',
    startDateAfterEndDateError: 'Начало должно быть раньше окончания.',
};

export type SchedulerLocaleBundle = {
    dateLocale: DateFnsLocale;
    localeText?: typeof localeTextRu;
};

export function schedulerLocaleFor(mmLocale: string): SchedulerLocaleBundle {
    if (localeLang(mmLocale) === 'ru') {
        return {dateLocale: dateFnsRuTitles, localeText: localeTextRu};
    }
    return {dateLocale: dateFnsEnUS};
}
