import React from 'react';

import Link from '@mui/material/Link';

const LINK_OR_EMAIL_RE = /(https?:\/\/[^\s]+|[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,})/gi;

function splitTrailingPunctuation(s: string): [string, string] {
    let cut = s.length;
    while (cut > 0 && ".,;:!?)']".includes(s[cut - 1]!)) {
        cut--;
    }
    return [s.slice(0, cut), s.slice(cut)];
}

/** Turn bare http(s) URLs and emails in plain text into clickable React nodes. */
export function linkifyTextNodes(text: string): React.ReactNode[] {
    if (!text) {
        return [];
    }
    const nodes: React.ReactNode[] = [];
    let last = 0;
    let key = 0;
    const re = new RegExp(LINK_OR_EMAIL_RE.source, 'gi');
    let m: RegExpExecArray | null;
    while ((m = re.exec(text)) !== null) {
        if (m.index > last) {
            nodes.push(text.slice(last, m.index));
        }
        const [linkValue, trailing] = splitTrailingPunctuation(m[0]);
        if (linkValue) {
            const isHttp = /^https?:\/\//i.test(linkValue);
            nodes.push(
                <Link
                    key={`lnk-${key++}`}
                    href={isHttp ? linkValue : `mailto:${linkValue}`}
                    target={isHttp ? '_blank' : undefined}
                    rel={isHttp ? 'noopener noreferrer' : undefined}
                    underline='hover'
                    sx={{wordBreak: 'break-all'}}
                >
                    {linkValue}
                </Link>,
            );
        }
        if (trailing) {
            nodes.push(trailing);
        }
        last = m.index + m[0].length;
    }
    if (last < text.length) {
        nodes.push(text.slice(last));
    }
    return nodes;
}
