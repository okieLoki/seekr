document.addEventListener('DOMContentLoaded', () => {

    const $ = id => document.getElementById(id);

    // Stats
    const statTotalDocs   = $('statTotalDocs');
    const statTotalTokens = $('statTotalTokens');

    // List
    const docList    = $('docList');
    const docRange   = $('docRange');
    const prevPage   = $('prevPage');
    const nextPage   = $('nextPage');
    const pageLabel  = $('pageLabel');
    const perPageSelect = $('perPageSelect');
    const searchInput = $('searchInput');

    // Modals
    const addModal      = $('addModal');
    const addDocIdLabel = $('addDocIdLabel');
    const addDocText    = $('addDocText');
    const indexDocBtn   = $('indexDocBtn');
    const bulkModal     = $('bulkModal');
    const bulkDocText   = $('bulkDocText');
    const submitBulkBtn = $('submitBulkBtn');
    const editModal     = $('editModal');
    const editDocIdLabel = $('editDocIdLabel');
    const editDocText   = $('editDocText');
    const saveEditBtn   = $('saveEditBtn');
    const toast         = $('toast');

    let debounce, page = 1, limit = 20, isSearch = false, editId = null;
    // ── Helpers ──────────────────────────────────────────────────────────────
    function esc(s) {
        return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
    }

    function showToast(msg, err = false) {
        toast.textContent = msg;
        toast.style.background = err ? '#dc2626' : '#16a34a';
        toast.classList.add('show');
        setTimeout(() => toast.classList.remove('show'), 3500);
    }

    // ── JSON syntax renderer ─────────────────────────────────────────────────
    function renderJson(val, depth = 0) {
        const pad = '  '.repeat(depth);
        const padC = '  '.repeat(Math.max(0, depth - 1));

        if (val === null)            return `<span class="jl">null</span>`;
        if (typeof val === 'boolean') return `<span class="jb">${val}</span>`;
        if (typeof val === 'number')  return `<span class="jn">${val}</span>`;
        if (typeof val === 'string')  return `<span class="js">"${esc(val)}"</span>`;

        if (Array.isArray(val)) {
            if (!val.length) return `<span class="jp">[]</span>`;
            const rows = val.map(v => `${pad}  ${renderJson(v, depth + 1)}`);
            return `<span class="jp">[</span>\n${rows.join('<span class="jp">,</span>\n')}\n${padC}<span class="jp">]</span>`;
        }

        if (typeof val === 'object') {
            const keys = Object.keys(val);
            if (!keys.length) return `<span class="jp">{}</span>`;
            const rows = keys.map(k =>
                `${pad}  <span class="jk">"${esc(k)}"</span><span class="jp">: </span>${renderJson(val[k], depth + 1)}`
            );
            return `<span class="jp">{</span>\n${rows.join('<span class="jp">,</span>\n')}\n${padC}<span class="jp">}</span>`;
        }
        return esc(String(val));
    }

    function syntaxHL(text) {
        try { return renderJson(JSON.parse(text)); }
        catch { return esc(text); }
    }

    // ── Card factory ─────────────────────────────────────────────────────────
    function buildCard(doc) {
        const card = document.createElement('div');
        card.className = 'doc-card';

        // Short ID display
        const shortId = doc.id.length > 24
            ? doc.id.slice(0, 8) + '…' + doc.id.slice(-4)
            : doc.id;

        // Detect if JSON and build field preview
        let isJSON = false;
        let parsed = null;
        let fieldHtml = '';
        let typeLabel = 'text';

        try {
            parsed = JSON.parse(doc.text);
            isJSON = typeof parsed === 'object' && parsed !== null;
        } catch {}

        if (isJSON && !Array.isArray(parsed)) {
            typeLabel = 'json';
            const entries = Object.entries(parsed)
                .filter(([, v]) => typeof v === 'string' || typeof v === 'number')
                .slice(0, 6);

            fieldHtml = entries.map(([k, v]) => {
                const val = String(v);
                const display = val.length > 50 ? val.slice(0, 50) + '…' : val;
                return `<span class="doc-field"><span class="field-key">${esc(k)}:</span><span class="field-val">"${esc(display)}"</span></span>`;
            }).join('');

            const remaining = Object.keys(parsed).length - entries.length;
            if (remaining > 0) fieldHtml += `<span class="field-more">+${remaining} more</span>`;
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
                <div class="doc-head-actions">
                    <button class="doc-edit-btn">Edit</button>
                </div>
            </div>
            <div class="doc-body">
                <pre class="json-view">${syntaxHL(doc.text)}</pre>
            </div>`;

        card.querySelector('.doc-head').addEventListener('click', e => {
            if (e.target.closest('.doc-head-actions')) return;
            card.classList.toggle('expanded');
        });

        card.querySelector('.doc-edit-btn').addEventListener('click', e => {
            e.stopPropagation();
            editId = doc.id;
            editDocIdLabel.textContent = doc.id;
            editDocText.value = doc.text;
            editModal.showModal();
        });

        return card;
    }

    function renderDocs(docs) {
        docList.innerHTML = '';
        if (!docs || !docs.length) {
            docList.innerHTML = '<div class="empty-state">No documents found.</div>';
            return;
        }
        docs.forEach(d => docList.appendChild(buildCard(d)));
    }

    // ── Stats ────────────────────────────────────────────────────────────────
    async function fetchStats() {
        try {
            const d = await fetch('/api/stats').then(r => r.json());
            statTotalDocs.textContent   = d.totalDocs.toLocaleString();
            statTotalTokens.textContent = d.totalLength.toLocaleString();
        } catch {}
    }

    // ── Documents ─────────────────────────────────────────────────────────────
    async function fetchDocs(p = 1) {
        try {
            const d = await fetch(`/api/documents?page=${p}&limit=${limit}`).then(r => r.json());
            renderDocs(d.documents || []);
            const s = (p - 1) * limit + 1, e = Math.min(p * limit, d.total);
            docRange.textContent = `${s}–${e} of ${d.total.toLocaleString()} documents`;
            pageLabel.textContent = `Page ${p}`;
            prevPage.disabled = p <= 1;
            nextPage.disabled = p * limit >= d.total;
        } catch {}
    }

    function refresh() { if (isSearch) return; fetchStats(); fetchDocs(page); }

    prevPage.addEventListener('click', () => { if (page > 1) fetchDocs(--page); });
    nextPage.addEventListener('click', () => fetchDocs(++page));
    perPageSelect.addEventListener('change', () => { limit = +perPageSelect.value; page = 1; fetchDocs(1); });

    // ── Search ────────────────────────────────────────────────────────────────
    searchInput.addEventListener('input', e => {
        clearTimeout(debounce);
        const q = e.target.value.trim();
        if (!q) {
            isSearch = false;
            prevPage.disabled = false;
            refresh();
            return;
        }
        isSearch = true;
        debounce = setTimeout(async () => {
            try {
                const d = await fetch(`/search?q=${encodeURIComponent(q)}`).then(r => r.json());
                const res = d.results || [];
                renderDocs(res);
                docRange.textContent = `${res.length} result${res.length !== 1 ? 's' : ''}`;
                pageLabel.textContent = 'Search';
                prevPage.disabled = nextPage.disabled = true;
            } catch {}
        }, 280);
    });

    // ── Add Modal ─────────────────────────────────────────────────────────────
    $('openAddModalBtn').addEventListener('click', () => {
        addDocIdLabel.textContent = crypto.randomUUID();
        addDocText.value = '';
        addModal.showModal();
    });
    $('cancelAddBtn').addEventListener('click', () => addModal.close());

    indexDocBtn.addEventListener('click', async () => {
        const text = addDocText.value.trim();
        if (!text) { showToast('Content cannot be empty.', true); return; }
        indexDocBtn.textContent = 'Indexing…'; indexDocBtn.disabled = true;
        try {
            const res = await fetch('/index', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ id: addDocIdLabel.textContent, text }) });
            if (!res.ok) throw new Error(await res.text());
            showToast('Document indexed.');
            addModal.close(); refresh();
        } catch(err) { showToast(err.message, true); }
        finally { indexDocBtn.textContent = 'Index Document'; indexDocBtn.disabled = false; }
    });

    // ── Bulk Modal ────────────────────────────────────────────────────────────
    $('openBulkModalBtn').addEventListener('click', () => { bulkDocText.value = ''; bulkModal.showModal(); });
    $('cancelBulkBtn').addEventListener('click', () => bulkModal.close());

    submitBulkBtn.addEventListener('click', async () => {
        const raw = bulkDocText.value.trim();
        let payload;
        try { payload = JSON.parse(raw); if (!Array.isArray(payload)) throw 0; }
        catch { showToast('Must be a valid JSON array.', true); return; }
        submitBulkBtn.textContent = `Importing ${payload.length}…`; submitBulkBtn.disabled = true;
        try {
            const res = await fetch('/bulk-index', { method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify(payload) });
            if (!res.ok) throw new Error(await res.text());
            const r = await res.json();
            showToast(`Imported ${r.indexed} doc${r.indexed !== 1 ? 's' : ''}${r.skipped ? `, skipped ${r.skipped}` : ''}.`, r.indexed === 0);
            bulkModal.close(); refresh();
        } catch(err) { showToast(err.message, true); }
        finally { submitBulkBtn.textContent = 'Import All'; submitBulkBtn.disabled = false; }
    });

    // ── Edit Modal ────────────────────────────────────────────────────────────
    $('cancelEditBtn').addEventListener('click', () => { editModal.close(); editId = null; });

    saveEditBtn.addEventListener('click', async () => {
        const text = editDocText.value.trim();
        if (!text) return;
        try {
            const res = await fetch(`/api/documents/update?id=${encodeURIComponent(editId)}`, { method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify({ text }) });
            if (!res.ok) throw new Error(await res.text());
            showToast('Document updated.');
            editModal.close();
            isSearch ? searchInput.dispatchEvent(new Event('input', {bubbles:true})) : refresh();
        } catch(err) { showToast(err.message, true); }
    });

    // ── Boot ──────────────────────────────────────────────────────────────────
    fetchStats();
    fetchDocs(1);
});
