// ponytail: theme color parsing for Mattermost CSS vars.
const assert = require('assert');

function mmCssColorFrom(raw, rgbFallback, fallback) {
    let v = (raw || '').trim();
    if (!v) {
        v = (rgbFallback || '').trim();
    }
    if (!v) {
        return fallback;
    }
    if (/^\d+(\s*,\s*\d+){2}/.test(v)) {
        return `rgb(${v})`;
    }
    return v;
}

function relativeLuminance(color) {
    const hex = color.match(/^#([0-9a-f]{3}|[0-9a-f]{6})$/i);
    let rgb;
    if (hex) {
        let h = hex[1];
        if (h.length === 3) {
            h = h.split('').map((c) => c + c).join('');
        }
        rgb = [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)];
    } else {
        const m = color.match(/rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i);
        rgb = m ? [Number(m[1]), Number(m[2]), Number(m[3])] : null;
    }
    if (!rgb) {
        return 1;
    }
    const lin = rgb.map((c) => {
        const s = c / 255;
        return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
    });
    return 0.2126 * lin[0] + 0.7152 * lin[1] + 0.0722 * lin[2];
}

assert.strictEqual(mmCssColorFrom('#1a1d21', '', '#fff'), '#1a1d21');
assert.strictEqual(mmCssColorFrom('26, 29, 33', '', '#fff'), 'rgb(26, 29, 33)');
assert.strictEqual(mmCssColorFrom('', '26, 29, 33', '#fff'), 'rgb(26, 29, 33)');
assert.ok(relativeLuminance('#1a1d21') < 0.45);
assert.ok(relativeLuminance('#ffffff') > 0.45);
console.log('mm_theme_check: ok');
