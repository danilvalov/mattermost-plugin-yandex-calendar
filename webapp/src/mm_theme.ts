import {createTheme, type Theme} from '@mui/material/styles';

/** Resolve Mattermost CSS color vars (hex or `r, g, b` / `--*-rgb`). */
export function mmCssColor(name: string, fallback: string, el: Element = document.body): string {
    const css = getComputedStyle(el);
    let raw = css.getPropertyValue(name).trim();
    if (!raw) {
        const rgbName = name.endsWith('-rgb') ? name : `${name}-rgb`;
        raw = css.getPropertyValue(rgbName).trim();
    }
    if (!raw) {
        return fallback;
    }
    if (/^\d+(\s*,\s*\d+){2}/.test(raw)) {
        return `rgb(${raw})`;
    }
    return raw;
}

function parseRGB(color: string): [number, number, number] | null {
    const hex = color.match(/^#([0-9a-f]{3}|[0-9a-f]{6})$/i);
    if (hex) {
        let h = hex[1];
        if (h.length === 3) {
            h = h.split('').map((c) => c + c).join('');
        }
        return [parseInt(h.slice(0, 2), 16), parseInt(h.slice(2, 4), 16), parseInt(h.slice(4, 6), 16)];
    }
    const rgb = color.match(/rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i);
    if (rgb) {
        return [Number(rgb[1]), Number(rgb[2]), Number(rgb[3])];
    }
    return null;
}

/** Relative luminance 0..1 (sRGB). */
export function relativeLuminance(color: string): number {
    const rgb = parseRGB(color);
    if (!rgb) {
        return 1;
    }
    const lin = rgb.map((c) => {
        const s = c / 255;
        return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
    });
    return 0.2126 * lin[0] + 0.7152 * lin[1] + 0.0722 * lin[2];
}

/**
 * Calendar product surface = Mattermost center-channel colors.
 * Always paint an opaque center-channel bg so denim `.main-wrapper` does not bleed through.
 */
export function createMattermostMuiTheme(): Theme {
    const root =
        document.querySelector('.product-wrapper') ||
        document.querySelector('#root') ||
        document.body;

    // Mattermost uses html { font-size: 62.5% } (10px). MUI rem math defaults to 16 —
    // without matching htmlFontSize every button/label shrinks to ~62% size.
    const htmlFontSize = parseFloat(getComputedStyle(document.documentElement).fontSize) || 16;

    const bg = mmCssColor('--center-channel-bg', '#ffffff', root);
    const text = mmCssColor('--center-channel-color', '#3f4350', root);
    const button = mmCssColor('--button-bg', '#1c58d9', root);
    const buttonColor = mmCssColor('--button-color', '#ffffff', root);
    const error = mmCssColor('--error-text', '#d24b4b', root);
    const link = mmCssColor('--link-color', button, root);
    const dark = relativeLuminance(bg) < 0.45;

    // Slightly lifted paper for dialogs/cards vs page bg.
    const paper = dark ? 'rgba(255,255,255,0.06)' : '#ffffff';
    // If bg is already near-white, paper stays white; if dark center-channel, blend up.
    const paperBg = dark
        ? (relativeLuminance(bg) < 0.2 ? '#2a2f3a' : bg)
        : (bg.toLowerCase() === '#ffffff' || bg === '#fff' || bg.startsWith('rgb(255') ? '#ffffff' : bg);

    return createTheme({
        palette: {
            mode: dark ? 'dark' : 'light',
            primary: {main: button, contrastText: buttonColor},
            secondary: {main: link},
            error: {main: error},
            background: {
                default: bg,
                paper: dark ? paperBg : paper,
            },
            text: {
                primary: text,
                secondary: dark ? 'rgba(255,255,255,0.72)' : 'rgba(63,67,80,0.72)',
            },
            divider: dark ? 'rgba(255,255,255,0.12)' : 'rgba(63,67,80,0.16)',
            action: {
                hover: dark ? 'rgba(255,255,255,0.08)' : 'rgba(63,67,80,0.06)',
                selected: dark ? 'rgba(255,255,255,0.12)' : 'rgba(63,67,80,0.08)',
            },
        },
        typography: {fontFamily: 'inherit', htmlFontSize},
        shape: {borderRadius: 8},
        components: {
            MuiPaper: {
                styleOverrides: {
                    root: {
                        backgroundImage: 'none',
                    },
                },
            },
            MuiButton: {
                styleOverrides: {
                    root: {
                        textTransform: 'none',
                    },
                    text: {
                        // Readable on both light and dark center-channel surfaces.
                        color: text,
                        '&:hover': {
                            backgroundColor: dark ? 'rgba(255,255,255,0.08)' : 'rgba(63,67,80,0.06)',
                        },
                    },
                },
            },
            MuiDialog: {
                styleOverrides: {
                    paper: {
                        backgroundImage: 'none',
                        backgroundColor: dark ? paperBg : '#ffffff',
                        boxShadow: dark
                            ? '0 12px 40px rgba(0,0,0,0.55)'
                            : '0 12px 40px rgba(0,0,0,0.18)',
                    },
                },
            },
            MuiIconButton: {
                styleOverrides: {
                    root: {
                        color: text,
                    },
                },
            },
        },
    });
}
