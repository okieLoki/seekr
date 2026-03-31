import { state, dom } from './state.js';
import { showToast, collectionParam } from './utils.js';
import { closeSidebar } from './sidebar.js';
import { refresh, triggerSearch } from './docs.js';
import { fetchCollections } from './collections.js';
import { queueImport } from './imports.js';

export function initModals() {
    document.getElementById('openAddModalBtn').addEventListener('click', () => {
        dom.addModalCollection.textContent = state.activeCollection;
        dom.addDocIdLabel.textContent = crypto.randomUUID();
        dom.addDocText.value = '';
        dom.addModal.showModal();
        closeSidebar();
    });

    document.getElementById('cancelAddBtn').addEventListener('click', () => dom.addModal.close());

    dom.indexDocBtn.addEventListener('click', async () => {
        const text = dom.addDocText.value.trim();
        if (!text) { showToast('Content cannot be empty.', true); return; }
        dom.indexDocBtn.textContent = 'Indexing…'; dom.indexDocBtn.disabled = true;
        try {
            const res = await fetch(`/index${collectionParam(state.activeCollection)}`, {
                method: 'POST', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ id: dom.addDocIdLabel.textContent, text })
            });
            if (!res.ok) throw new Error(await res.text());
            showToast('Document indexed.');
            dom.addModal.close();
            refresh();
            fetchCollections();
        } catch (err) { showToast(err.message, true); }
        finally { dom.indexDocBtn.textContent = 'Index Document'; dom.indexDocBtn.disabled = false; }
    });

    document.getElementById('openBulkModalBtn').addEventListener('click', () => {
        dom.bulkModalCollection.textContent = state.activeCollection;
        dom.bulkDocText.value = '';
        dom.bulkModal.showModal();
        closeSidebar();
    });

    document.getElementById('cancelBulkBtn').addEventListener('click', () => dom.bulkModal.close());

    dom.submitBulkBtn.addEventListener('click', async () => {
        let payload;
        try { payload = JSON.parse(dom.bulkDocText.value.trim()); if (!Array.isArray(payload)) throw 0; }
        catch { showToast('Must be a valid JSON array.', true); return; }
        dom.submitBulkBtn.textContent = `Queueing ${payload.length}…`; dom.submitBulkBtn.disabled = true;
        try {
            const job = await queueImport(payload);
            showToast(`Import queued: ${job.total} document${job.total !== 1 ? 's' : ''}.`);
            dom.bulkModal.close();
        } catch (err) { showToast(err.message, true); }
        finally { dom.submitBulkBtn.textContent = 'Import All'; dom.submitBulkBtn.disabled = false; }
    });

    document.getElementById('cancelEditBtn').addEventListener('click', () => { dom.editModal.close(); state.editId = null; });

    dom.saveEditBtn.addEventListener('click', async () => {
        const text = dom.editDocText.value.trim();
        if (!text) return;
        try {
            const res = await fetch(`/api/documents/update${collectionParam(state.activeCollection)}&id=${encodeURIComponent(state.editId)}`, {
                method: 'PUT', headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ text })
            });
            if (!res.ok) throw new Error(await res.text());
            showToast('Document updated.');
            dom.editModal.close();
            state.isSearch ? triggerSearch() : refresh();
        } catch (err) { showToast(err.message, true); }
    });
}
