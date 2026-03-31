import { esc } from './utils.js';

function highlightJsonLine(line) {
    return esc(line)
        .replace(/&quot;([^&]*)&quot;(?=\s*:)/g, '<span class="json-key">"$1"</span>')
        .replace(/:\s*&quot;([^&]*)&quot;/g, ': <span class="json-string">"$1"</span>')
        .replace(/: (-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)/g, ': <span class="json-number">$1</span>')
        .replace(/: (true|false)\b/g, ': <span class="json-boolean">$1</span>')
        .replace(/: null\b/g, ': <span class="json-null">null</span>');
}

export function renderJsonDocument(text) {
    try {
        const parsed = JSON.parse(text);
        const formatted = JSON.stringify(parsed, null, 2);
        const lines = formatted.split('\n');
        return `
            <div class="json-pretty">
                ${lines.map((line, index) => `
                    <div class="json-line">
                        <span class="json-line-no">${index + 1}</span>
                        <code class="json-line-code">${highlightJsonLine(line)}</code>
                    </div>
                `).join('')}
            </div>
        `;
    } catch {
        return `<pre class="json-fallback">${esc(text)}</pre>`;
    }
}
