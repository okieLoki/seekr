import { state, dom } from './state.js';
import { esc, showToast, collectionParam } from './utils.js';
import { renderBoostRows } from './boost.js';
import { getBoosts } from './boost.js';
import { renderJsonDocument } from './jsonview.js';

function buildDocBody(text, isJSON) {
    return isJSON ? renderJsonDocument(text) : `<pre class="json-fallback">${esc(text)}</pre>`;
}

function updateFieldSuggestions(docs) {
    const fields = new Set(state.knownFields);
    docs.forEach(doc => {
        try {
            const parsed = JSON.parse(doc.text);
            if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
                Object.keys(parsed).forEach(k => fields.add(k));
            }
        } catch { }
    });
    state.knownFields = [...fields];
    dom.fieldSuggestions.innerHTML = state.knownFields.map(f => `<option value="${esc(f)}">`).join('');
    renderBoostRows();
}

function buildCard(doc) {
    const card = document.createElement('div');
    card.className = 'doc-card';
    const shortId = doc.id.length > 24 ? doc.id.slice(0, 8) + '…' + doc.id.slice(-4) : doc.id;
    let isJSON = false, parsed = null, fieldHtml = '', typeLabel = 'text';
    try { parsed = JSON.parse(doc.text); isJSON = typeof parsed === 'object' && parsed !== null; } catch { }
    if (isJSON && !Array.isArray(parsed)) {
        typeLabel = 'json';
        const entries = Object.entries(parsed).filter(([, v]) => typeof v === 'string' || typeof v === 'number').slice(0, 6);
        fieldHtml = entries.map(([k, v]) => {
            const d = String(v).length > 50 ? String(v).slice(0, 50) + '…' : String(v);
            return `<span class="doc-field"><span class="field-key">${esc(k)}:</span><span class="field-val">"${esc(d)}"</span></span>`;
        }).join('');
        const rem = Object.keys(parsed).length - entries.length;
        if (rem > 0) fieldHtml += `<span class="field-more">+${rem} more</span>`;
    } else if (Array.isArray(parsed)) {
        typeLabel = 'array';
        fieldHtml = `<span class="doc-plain-preview">[Array · ${parsed.length} items]</span>`;
    } else {
        const preview = doc.text.length > 160 ? doc.text.slice(0, 160) + '…' : doc.text;
        fieldHtml = `<span class="doc-plain-preview">${esc(preview)}</span>`;
    }
    card.innerHTML = `
        <div class="doc-head">
            <svg class="doc-expand-btn" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="9 18 15 12 9 6"/></svg>
            <div class="doc-head-content">
                <div class="doc-id-row">
                    <code class="doc-id-badge" title="${esc(doc.id)}">${esc(shortId)}</code>
                    <span class="doc-type-badge">${typeLabel}</span>
                </div>
                <div class="doc-fields">${fieldHtml}</div>
            </div>
            <div class="doc-head-actions"><button class="doc-edit-btn">Edit</button></div>
        </div>
        <div class="doc-body"></div>`;
    const bodyEl = card.querySelector('.doc-body');
    card.querySelector('.doc-head').addEventListener('click', e => {
        if (e.target.closest('.doc-head-actions')) return;
        if (!card.classList.contains('expanded') && !bodyEl.dataset.rendered) {
            bodyEl.innerHTML = buildDocBody(doc.text, isJSON);
            bodyEl.dataset.rendered = 'true';
        }
        card.classList.toggle('expanded');
    });
    card.querySelector('.doc-edit-btn').addEventListener('click', e => {
        e.stopPropagation();
        state.editId = doc.id;
        dom.editDocIdLabel.textContent = doc.id;
        dom.editDocText.value = doc.text;
        dom.editModal.showModal();
    });
    return card;
}

function renderDocs(docs) {
    dom.docList.innerHTML = '';
    if (!docs.length) { dom.docList.innerHTML = '<div class="empty-state">No documents found.</div>'; return; }
    const fragment = document.createDocumentFragment();
    docs.forEach(d => fragment.appendChild(buildCard(d)));
    dom.docList.appendChild(fragment);
}

function abortRequest(key) {
    const current = state[key];
    if (current) {
        current.abort();
    }
    const controller = new AbortController();
    state[key] = controller;
    return controller;
}

export async function fetchStats() {
    if (!state.activeCollection) return;
    const controller = abortRequest('statsAbortController');
    const activeCollection = state.activeCollection;
    try {
        const res = await fetch(`/api/stats${collectionParam(activeCollection)}`, { signal: controller.signal });
        const d = await res.json();
        if (state.activeCollection !== activeCollection) return;
        dom.statTotalDocs.textContent = d.totalDocs.toLocaleString();
        dom.statTotalTokens.textContent = d.totalLength.toLocaleString();
    } catch (err) {
        if (err.name !== 'AbortError') {
            return;
        }
    } finally {
        if (state.statsAbortController === controller) {
            state.statsAbortController = null;
        }
    }
}

export async function fetchDocs(p = 1) {
    if (!state.activeCollection) return;
    const controller = abortRequest('docsAbortController');
    const activeCollection = state.activeCollection;
    const param = collectionParam(activeCollection);
    try {
        const res = await fetch(`/api/documents${param}&page=${p}&limit=${state.limit}`, { signal: controller.signal });
        const d = await res.json();
        if (state.activeCollection !== activeCollection) return;
        const docs = d.documents || [];
        renderDocs(docs);
        updateFieldSuggestions(docs);
        const s = d.total ? (p - 1) * state.limit + 1 : 0;
        const e = Math.min(p * state.limit, d.total || 0);
        dom.docRange.textContent = d.total ? `${s}–${e} of ${d.total.toLocaleString()} documents` : '0 documents';
        dom.pageLabel.textContent = `Page ${p}`;
        dom.prevPage.disabled = p <= 1;
        dom.nextPage.disabled = p * state.limit >= d.total;
    } catch (err) {
        if (err.name !== 'AbortError') {
            showToast('Unable to load documents right now.', true);
        }
    } finally {
        if (state.docsAbortController === controller) {
            state.docsAbortController = null;
        }
    }
}

export function refresh() {
    if (state.isSearch) return;
    fetchStats();
    fetchDocs(state.page);
}

export function triggerSearch() {
    clearTimeout(state.debounce);
    const q = dom.searchInput.value.trim();
    if (!q) {
        state.isSearch = false;
        if (state.searchAbortController) {
            state.searchAbortController.abort();
            state.searchAbortController = null;
        }
        dom.prevPage.disabled = dom.nextPage.disabled = false;
        refresh();
        return;
    }
    state.isSearch = true;
    state.debounce = setTimeout(async () => {
        const controller = abortRequest('searchAbortController');
        const activeCollection = state.activeCollection;
        try {
            const body = { q, boosts: getBoosts() };
            const res = await fetch(`/search${collectionParam(activeCollection)}`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body),
                signal: controller.signal
            });
            const d = await res.json();
            if (state.activeCollection !== activeCollection || dom.searchInput.value.trim() !== q) return;
            const results = d.results || [];
            renderDocs(results);
            updateFieldSuggestions(results);
            dom.docRange.textContent = `${results.length} result${results.length !== 1 ? 's' : ''}`;
            dom.pageLabel.textContent = 'Search';
            dom.prevPage.disabled = dom.nextPage.disabled = true;
        } catch (err) {
            if (err.name !== 'AbortError') {
                showToast('Search failed. Please try again.', true);
            }
        } finally {
            if (state.searchAbortController === controller) {
                state.searchAbortController = null;
            }
        }
    }, 280);
}

export function initDocs() {
    dom.searchInput.addEventListener('input', triggerSearch);
    dom.prevPage.addEventListener('click', () => { if (state.page > 1) fetchDocs(--state.page); });
    dom.nextPage.addEventListener('click', () => fetchDocs(++state.page));
    dom.perPageSelect.addEventListener('change', () => { state.limit = +dom.perPageSelect.value; state.page = 1; fetchDocs(1); });
}
