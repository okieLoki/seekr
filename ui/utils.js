import { dom } from './state.js';

export function esc(s) {
    return String(s)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

let toastTimer = null;

export function showToast(msg, err = false) {
    dom.toast.textContent = msg;
    dom.toast.style.background = err ? '#dc2626' : '#16a34a';
    dom.toast.classList.add('show');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => dom.toast.classList.remove('show'), 3500);
}

export function collectionParam(activeCollection) {
    return activeCollection ? `?collection=${encodeURIComponent(activeCollection)}` : '';
}

function renderJson(val, depth = 0) {
    const pad = '  '.repeat(depth);
    if (val === null) return `<span class="jl">null</span>`;
    if (typeof val === 'boolean') return `<span class="jb">${val}</span>`;
    if (typeof val === 'number') return `<span class="jn">${val}</span>`;
    if (typeof val === 'string') return `<span class="js">"${esc(val)}"</span>`;
    if (Array.isArray(val)) {
        if (!val.length) return `<span class="jp">[]</span>`;
        const rows = val.map(v => `${pad}  ${renderJson(v, depth + 1)}`);
        return `<span class="jp">[</span>\n${rows.join('<span class="jp">,</span>\n')}\n${pad}<span class="jp">]</span>`;
    }
    if (typeof val === 'object') {
        const keys = Object.keys(val);
        if (!keys.length) return `<span class="jp">{}</span>`;
        const rows = keys.map(k =>
            `${pad}  <span class="jk">"${esc(k)}"</span><span class="jp">: </span>${renderJson(val[k], depth + 1)}`
        );
        return `<span class="jp">{</span>\n${rows.join('<span class="jp">,</span>\n')}\n${pad}<span class="jp">}</span>`;
    }
    return esc(String(val));
}

export function syntaxHL(text) {
    try { return renderJson(JSON.parse(text)); } catch { return esc(text); }
}
