import { state, dom } from './state.js';
import { showToast, collectionParam } from './utils.js';
import { closeSidebar } from './sidebar.js';
import { refresh, triggerSearch } from './docs.js';
import { fetchCollections } from './collections.js';
import { queueImport } from './imports.js';
import { BULK_PASTE_CHAR_LIMIT, readBulkJsonFile, validateBulkJsonText } from './bulk.js';

function updateBulkCounter() {
    const len = dom.bulkDocText.value.length;
    dom.bulkCharCount.textContent = `${len.toLocaleString()} / ${BULK_PASTE_CHAR_LIMIT.toLocaleString()} chars`;
    dom.bulkCharCount.classList.toggle('danger', len > BULK_PASTE_CHAR_LIMIT * 0.9);
}

function showBulkValidation(message = '', isError = false) {
    dom.bulkValidation.textContent = message;
    dom.bulkValidation.classList.toggle('error', Boolean(message && isError));
    dom.bulkValidation.classList.toggle('success', Boolean(message && !isError));
}

async function handleBulkFileSelection() {
    const [file] = dom.bulkFileInput.files || [];
    if (!file) return;

    try {
        const result = await readBulkJsonFile(file);
        if (!result.ok) {
            showBulkValidation(result.error, true);
            showToast(result.error, true);
            return;
        }

        dom.bulkDocText.value = result.text;
        updateBulkCounter();
        showBulkValidation(`Loaded ${result.payload.length} item${result.payload.length !== 1 ? 's' : ''} from ${file.name}.`);
    } catch {
        showBulkValidation('Unable to read that file.', true);
        showToast('Unable to read that file.', true);
    } finally {
        dom.bulkFileInput.value = '';
    }
}

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
        dom.bulkFileInput.value = '';
        updateBulkCounter();
        showBulkValidation('');
        dom.bulkModal.showModal();
        closeSidebar();
    });

    document.getElementById('cancelBulkBtn').addEventListener('click', () => dom.bulkModal.close());
    dom.bulkDocText.addEventListener('input', () => {
        updateBulkCounter();
        showBulkValidation('');
    });
    dom.bulkFileInput.addEventListener('change', handleBulkFileSelection);

    dom.submitBulkBtn.addEventListener('click', async () => {
        const result = validateBulkJsonText(dom.bulkDocText.value);
        if (!result.ok) {
            showBulkValidation(result.error, true);
            showToast(result.error, true);
            return;
        }
        const payload = result.payload;
        dom.submitBulkBtn.textContent = `Queueing ${payload.length}…`; dom.submitBulkBtn.disabled = true;
        try {
            const job = await queueImport(payload);
            showToast(`Import queued: ${job.total} document${job.total !== 1 ? 's' : ''}.`);
            showBulkValidation('');
            dom.bulkModal.close();
        } catch (err) { showToast(err.message, true); }
        finally { dom.submitBulkBtn.textContent = 'Import All'; dom.submitBulkBtn.disabled = false; }
    });

    updateBulkCounter();

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
