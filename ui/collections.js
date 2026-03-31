import { state, dom } from './state.js';
import { esc, showToast } from './utils.js';
import { closeSidebar } from './sidebar.js';

export function closeMenu() {
    if (state.openMenuEl) { state.openMenuEl.remove(); state.openMenuEl = null; }
}

function showCollMenu(name, btn) {
    closeMenu();
    btn.classList.add('open');
    const rect = btn.getBoundingClientRect();
    const menu = document.createElement('div');
    menu.className = 'coll-dropdown';
    menu.style.cssText = `position:fixed;top:${rect.bottom + 4}px;left:${rect.left - 100}px`;
    menu.innerHTML = `
        <button class="coll-dropdown-item" data-action="rename">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
            Rename
        </button>
        <button class="coll-dropdown-item" data-action="duplicate">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
            Duplicate
        </button>
        <div class="coll-dropdown-sep"></div>
        <button class="coll-dropdown-item danger" data-action="delete">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6"/><path d="M14 11v6"/></svg>
            Delete
        </button>`;
    menu.querySelector('[data-action="rename"]').addEventListener('click', () => { closeMenu(); renameCollection(name); });
    menu.querySelector('[data-action="duplicate"]').addEventListener('click', () => { closeMenu(); duplicateCollection(name); });
    menu.querySelector('[data-action="delete"]').addEventListener('click', () => { closeMenu(); deleteCollection(name); });
    document.body.appendChild(menu);
    state.openMenuEl = menu;
    setTimeout(() => {
        const r = menu.getBoundingClientRect();
        if (r.right > window.innerWidth) menu.style.left = `${rect.right - r.width}px`;
        if (r.bottom > window.innerHeight) menu.style.top = `${rect.top - r.height - 4}px`;
    }, 0);
}

async function getAllDocs(collection) {
    const d = await fetch(`/api/documents?collection=${encodeURIComponent(collection)}&page=1&limit=10000`).then(r => r.json());
    return (d.documents || []).map(doc => ({ id: doc.id, text: doc.text }));
}

async function renameCollection(oldName) {
    const newName = prompt('Rename collection to:', oldName);
    if (!newName || newName.trim() === oldName) return;
    const trimmed = newName.trim();
    const createRes = await fetch('/api/collections', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: trimmed })
    });
    if (!createRes.ok) { showToast(await createRes.text(), true); return; }
    const docs = await getAllDocs(oldName);
    if (docs.length) {
        await fetch(`/bulk-index?collection=${encodeURIComponent(trimmed)}`, {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(docs)
        });
    }
    await fetch(`/api/collections?name=${encodeURIComponent(oldName)}`, { method: 'DELETE' });
    showToast(`Renamed to "${trimmed}".`);
    if (state.activeCollection === oldName) {
        await switchCollection(trimmed);
    } else {
        await fetchCollections();
    }
}

async function duplicateCollection(srcName) {
    const newName = prompt('Duplicate as:', srcName + '_copy');
    if (!newName || !newName.trim()) return;
    const trimmed = newName.trim();
    const createRes = await fetch('/api/collections', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: trimmed })
    });
    if (!createRes.ok) { showToast(await createRes.text(), true); return; }
    const docs = await getAllDocs(srcName);
    if (docs.length) {
        const res = await fetch(`/bulk-index?collection=${encodeURIComponent(trimmed)}`, {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(docs)
        });
        const r = await res.json();
        showToast(`Duplicated: ${r.indexed} doc${r.indexed !== 1 ? 's' : ''} copied to "${trimmed}".`);
    } else {
        showToast(`"${trimmed}" created (empty).`);
    }
    await fetchCollections();
}

export function showNoCollectionState() {
    state.activeCollection = null;
    dom.bcCollection.textContent = '—';
    dom.statTotalDocs.textContent = '—';
    dom.statTotalTokens.textContent = '—';
    dom.docList.innerHTML = '<div class="empty-state">No collections yet.<br>Create one above to get started.</div>';
    dom.docRange.textContent = '';
    dom.pageLabel.textContent = '';
    dom.prevPage.disabled = true;
    dom.nextPage.disabled = true;
}

export async function fetchCollections() {
    const d = await fetch('/api/collections').then(r => r.json());
    const cols = d.collections || [];
    dom.collectionList.innerHTML = '';
    if (!cols.length) {
        showNoCollectionState();
        return;
    }
    if (!state.activeCollection) {
        return switchCollection(cols[0].name);
    }
    const fragment = document.createDocumentFragment();
    cols.forEach(c => {
        const el = document.createElement('div');
        el.className = 'collection-item' + (c.name === state.activeCollection ? ' active' : '');
        el.innerHTML = `
            <span class="coll-icon">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/></svg>
            </span>
            <span class="coll-name">${esc(c.name)}</span>
            <span class="coll-count">${c.totalDocs.toLocaleString()}</span>
            <button class="coll-menu-btn" title="Options">⋯</button>`;
        el.querySelector('.coll-name').addEventListener('click', () => { switchCollection(c.name); closeSidebar(); });
        el.querySelector('.coll-icon').addEventListener('click', () => { switchCollection(c.name); closeSidebar(); });
        el.querySelector('.coll-menu-btn').addEventListener('click', e => { e.stopPropagation(); showCollMenu(c.name, e.currentTarget); });
        fragment.appendChild(el);
    });
    dom.collectionList.appendChild(fragment);
}

export async function switchCollection(name) {
    state.activeCollection = name;
    dom.bcCollection.textContent = name;
    state.isSearch = false;
    dom.searchInput.value = '';
    state.knownFields = [];
    state.page = 1;
    await fetchCollections();
    window._onCollectionChange && window._onCollectionChange();
    window._refresh();
}

async function deleteCollection(name) {
    if (!confirm(`Delete collection "${name}" and all its documents? This cannot be undone.`)) return;
    const res = await fetch(`/api/collections?name=${encodeURIComponent(name)}`, { method: 'DELETE' });
    if (!res.ok) { showToast(await res.text(), true); return; }
    showToast(`Collection "${name}" deleted.`);
    const colsRes = await fetch('/api/collections').then(r => r.json());
    const remaining = (colsRes.collections || []).map(c => c.name);
    if (!remaining.length) {
        showNoCollectionState();
        await fetchCollections();
        return;
    }
    const next = name === state.activeCollection ? remaining[0] : state.activeCollection;
    await switchCollection(next);
}

export function initCollections() {
    document.addEventListener('click', e => {
        if (state.openMenuEl && !state.openMenuEl.contains(e.target) && !e.target.closest('.coll-menu-btn')) {
            closeMenu();
        }
    }, true);

    document.getElementById('newCollectionBtn').addEventListener('click', () => {
        dom.newCollectionName.value = '';
        dom.newCollectionModal.showModal();
        setTimeout(() => dom.newCollectionName.focus(), 50);
    });

    document.getElementById('cancelNewCollectionBtn').addEventListener('click', () => dom.newCollectionModal.close());

    dom.createCollectionBtn.addEventListener('click', async () => {
        const name = dom.newCollectionName.value.trim();
        if (!name) return;
        const res = await fetch('/api/collections', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name })
        });
        if (!res.ok) { showToast(await res.text(), true); return; }
        showToast(`Collection "${name}" created.`);
        dom.newCollectionModal.close();
        await switchCollection(name);
    });

    dom.newCollectionName.addEventListener('keydown', e => { if (e.key === 'Enter') dom.createCollectionBtn.click(); });
}
