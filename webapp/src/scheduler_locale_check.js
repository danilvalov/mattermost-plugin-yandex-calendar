// ponytail: ru month titles must be nominative + capitalized (MUI header uses MMMM).
const assert = require('assert');
const {format} = require('date-fns');
const {ru} = require('date-fns/locale/ru');

const RU_MONTHS_NOM = [
    'Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь',
    'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь',
];

const dateFnsRuTitles = {
    ...ru,
    localize: {
        ...ru.localize,
        month: (n, options) => {
            if ((options?.width ?? 'wide') === 'wide') {
                return RU_MONTHS_NOM[n];
            }
            return ru.localize.month(n, options);
        },
    },
};

const d = new Date(2026, 6, 27);
assert.strictEqual(format(d, 'MMMM yyyy', {locale: dateFnsRuTitles}), 'Июль 2026');
assert.notStrictEqual(format(d, 'MMMM yyyy', {locale: ru}), 'Июль 2026'); // stock is genitive
console.log('scheduler_locale_check: ok');
