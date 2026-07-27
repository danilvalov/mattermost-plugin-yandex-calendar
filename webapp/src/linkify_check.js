const assert = require('assert');

// Mirrors webapp/src/linkify.tsx URL/email split (ponytail: keep in sync).
const LINK_OR_EMAIL_RE = /(https?:\/\/[^\s]+|[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,})/gi;

function splitTrailingPunctuation(s) {
    let cut = s.length;
    while (cut > 0 && ".,;:!?)']".includes(s[cut - 1])) {
        cut--;
    }
    return [s.slice(0, cut), s.slice(cut)];
}

function linkifyParts(text) {
    const parts = [];
    let last = 0;
    const re = new RegExp(LINK_OR_EMAIL_RE.source, 'gi');
    let m;
    while ((m = re.exec(text)) !== null) {
        if (m.index > last) {
            parts.push({type: 'text', v: text.slice(last, m.index)});
        }
        const [linkValue, trailing] = splitTrailingPunctuation(m[0]);
        parts.push({type: 'link', v: linkValue});
        if (trailing) {
            parts.push({type: 'text', v: trailing});
        }
        last = m.index + m[0].length;
    }
    if (last < text.length) {
        parts.push({type: 'text', v: text.slice(last)});
    }
    return parts;
}

const parts = linkifyParts('Join\nhttps://telemost.360.yandex.ru/j/8510081139.\nmail: a@b.co');
assert.strictEqual(parts.some((p) => p.type === 'link' && p.v === 'https://telemost.360.yandex.ru/j/8510081139'), true);
assert.strictEqual(parts.some((p) => p.type === 'text' && p.v === '.'), true);
assert.strictEqual(parts.some((p) => p.type === 'link' && p.v === 'a@b.co'), true);
console.log('linkify_check ok');
