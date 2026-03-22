document.addEventListener('DOMContentLoaded', () => {

    // Database Elements
    const statTotalDocs = document.getElementById('statTotalDocs');
    const statTotalTokens = document.getElementById('statTotalTokens');
    const docsTableBody = document.getElementById('docsTableBody');
    const prevPageBtn = document.getElementById('prevPage');
    const nextPageBtn = document.getElementById('nextPage');
    const pageInfo = document.getElementById('pageInfo');
    const searchInput = document.getElementById('searchInput');

    // Add Elements
    const openAddModalBtn = document.getElementById('openAddModalBtn');
    const addModal = document.getElementById('addModal');
    const addDocIdLabel = document.getElementById('addDocIdLabel');
    const addDocText = document.getElementById('addDocText');
    const cancelAddBtn = document.getElementById('cancelAddBtn');
    const indexDocBtn = document.getElementById('indexDocBtn');

    // Edit Elements
    const editModal = document.getElementById('editModal');
    const editDocIdLabel = document.getElementById('editDocIdLabel');
    const editDocText = document.getElementById('editDocText');
    const cancelEditBtn = document.getElementById('cancelEditBtn');
    const saveEditBtn = document.getElementById('saveEditBtn');
    
    // Core Elements
    const toast = document.getElementById('toast');

    let debounceTimer;
    let currentPage = 1;
    let currentLimit = 10;
    let currentEditDocId = null;

    function showToast(msg, isError = false) {
        toast.textContent = msg;
        toast.style.background = isError ? '#ef4444' : '#10b981';
        toast.classList.add('show');
        setTimeout(() => toast.classList.remove('show'), 3000);
    }

    function renderDocs(docs) {
        docsTableBody.innerHTML = '';
        if (!docs || docs.length === 0) {
            docsTableBody.innerHTML = '<tr><td colspan="3" style="text-align:center;color:#64748b;padding:2rem;">No documents available.</td></tr>';
            return;
        }

        docs.forEach(doc => {
            const tr = document.createElement('tr');
            
            const tdId = document.createElement('td');
            const idSpan = document.createElement('span');
            idSpan.className = 'code-font';
            idSpan.style.fontSize = '0.8rem';
            idSpan.style.color = 'var(--text-secondary)';
            idSpan.title = doc.id;
            idSpan.textContent = doc.id.length > 8 ? doc.id.slice(0, 8) + '...' : doc.id;
            tdId.appendChild(idSpan);
            
            const tdText = document.createElement('td');
            const span = document.createElement('span');
            span.className = 'doc-snippet';
            span.textContent = doc.text;
            tdText.appendChild(span);
            
            const tdActions = document.createElement('td');
            const editBtn = document.createElement('button');
            editBtn.className = 'btn btn-secondary btn-sm';
            editBtn.textContent = 'Edit';
            editBtn.onclick = () => openEditModal(doc.id, doc.text);
            tdActions.appendChild(editBtn);

            tr.appendChild(tdId);
            tr.appendChild(tdText);
            tr.appendChild(tdActions);
            docsTableBody.appendChild(tr);
        });
    }

    // ========== SEARCH LOGIC ==========
    searchInput.addEventListener('input', (e) => {
        const query = e.target.value.trim();
        clearTimeout(debounceTimer);
        
        if (!query) {
            refreshDatabaseView();
            return;
        }

        debounceTimer = setTimeout(async () => {
            try {
                const res = await fetch(`/search?q=${encodeURIComponent(query)}`);
                const data = await res.json();
                renderDocs(data.results || []);
                pageInfo.textContent = 'Search Mode';
                prevPageBtn.disabled = true;
                nextPageBtn.disabled = true;
            } catch(err) {
                console.error(err);
                docsTableBody.innerHTML = '<tr><td colspan="3" style="text-align:center;color:#ef4444;">Search failed</td></tr>';
            }
        }, 300);
    });

    // ========== DATABASE LOGIC ==========
    async function fetchStats() {
        try {
            const res = await fetch('/api/stats');
            const data = await res.json();
            statTotalDocs.textContent = data.totalDocs;
            statTotalTokens.textContent = data.totalLength;
        } catch(e) {
            console.error(e);
        }
    }

    async function fetchDocuments(page = 1) {
        if (searchInput.value.trim() !== '') return;
        try {
            const res = await fetch(`/api/documents?page=${page}&limit=${currentLimit}`);
            const data = await res.json();
            
            renderDocs(data.documents || []);

            pageInfo.textContent = `Page ${page}`;
            prevPageBtn.disabled = page <= 1;
            nextPageBtn.disabled = (page * currentLimit) >= data.total;
        } catch(e) {
            console.error(e);
        }
    }

    function refreshDatabaseView() {
        if (searchInput.value.trim() !== '') return;
        fetchStats();
        fetchDocuments(currentPage);
    }

    prevPageBtn.addEventListener('click', () => {
        if (currentPage > 1) {
            currentPage--;
            fetchDocuments(currentPage);
        }
    });

    nextPageBtn.addEventListener('click', () => {
        currentPage++;
        fetchDocuments(currentPage);
    });

    // ========== ADD MODAL ==========
    openAddModalBtn.addEventListener('click', () => {
        addDocIdLabel.textContent = crypto.randomUUID();
        addDocText.value = '';
        addModal.showModal();
    });

    cancelAddBtn.addEventListener('click', () => addModal.close());

    indexDocBtn.addEventListener('click', async () => {
        const id = addDocIdLabel.textContent;
        const text = addDocText.value.trim();

        if (!text) {
            showToast('Payload cannot be empty.', true);
            return;
        }

        const btnOrig = indexDocBtn.textContent;
        indexDocBtn.textContent = 'Indexing...';
        indexDocBtn.disabled = true;

        try {
            const res = await fetch('/index', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ id, text })
            });

            if (!res.ok) throw new Error(await res.text());

            showToast('Document successfully indexed natively.');
            addModal.close();
            refreshDatabaseView();
        } catch (err) {
            showToast(err.message, true);
        } finally {
            indexDocBtn.textContent = btnOrig;
            indexDocBtn.disabled = false;
        }
    });

    // ========== EDIT MODAL ==========
    function openEditModal(id, text) {
        currentEditDocId = id;
        editDocIdLabel.textContent = id;
        editDocText.value = text;
        editModal.showModal();
    }

    cancelEditBtn.addEventListener('click', () => {
        editModal.close();
        currentEditDocId = null;
    });

    saveEditBtn.addEventListener('click', async () => {
        const text = editDocText.value.trim();
        if (!text) return;

        try {
            const res = await fetch(`/api/documents/update?id=${currentEditDocId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ text })
            });

            if (!res.ok) throw new Error(await res.text());

            showToast('Document successfully updated!');
            editModal.close();
            
            // If in search mode, trigger search update. If in DB mode, fresh DB.
            if (searchInput.value.trim() !== '') {
                searchInput.dispatchEvent(new Event('input', { bubbles: true }));
            } else {
                refreshDatabaseView();
            }
        } catch (err) {
            showToast(err.message, true);
        }
    });

    // Boot routines
    fetchStats();
    refreshDatabaseView();
});
