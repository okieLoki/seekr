import { state, dom } from './state.js';
import { showToast, collectionParam } from './utils.js';
import { fetchCollections } from './collections.js';
import { refresh } from './docs.js';

function statusLabel(status) {
    switch (status) {
        case 'queued': return 'Queued';
        case 'processing': return 'Processing';
        case 'completed': return 'Completed';
        case 'failed': return 'Failed';
        default: return status;
    }
}

function progressPct(job) {
    if (!job.total) return 0;
    return Math.max(0, Math.min(100, Math.round((job.processed / job.total) * 100)));
}

function sortJobs(jobs) {
    return [...jobs].sort((a, b) => b.createdAt - a.createdAt);
}

function renderImportJobs() {
    const jobs = sortJobs(state.importJobs).slice(0, 6);
    const active = jobs.filter(j => j.status === 'queued' || j.status === 'processing');
    const dismissible = jobs.length > 0 && active.length === 0;

    dom.dismissImportPanelBtn.classList.toggle('hidden', !dismissible);
    dom.importPanel.classList.toggle('hidden', jobs.length === 0 || (state.importPanelDismissed && active.length === 0));
    if (!jobs.length) {
        dom.importPanelSummary.textContent = 'No active imports';
        dom.importList.innerHTML = '';
        return;
    }

    dom.importPanelSummary.textContent = active.length
        ? `${active.length} active import${active.length !== 1 ? 's' : ''}`
        : 'Recent import results';

    dom.importList.innerHTML = jobs.map(job => {
        const pct = progressPct(job);
        const meta = `${job.indexed} indexed${job.skipped ? ` • ${job.skipped} skipped` : ''}`;
        return `
            <article class="import-card import-${job.status}">
                <div class="import-card-head">
                    <div>
                        <div class="import-card-title">${statusLabel(job.status)}</div>
                        <div class="import-card-meta">${job.collection} • ${meta}</div>
                    </div>
                    <div class="import-card-count">${job.processed}/${job.total}</div>
                </div>
                <div class="import-progress">
                    <span class="import-progress-bar" style="width:${pct}%"></span>
                </div>
                <div class="import-card-foot">
                    <span>${pct}% processed</span>
                    <span>${job.error || (job.errors?.[0] ?? '')}</span>
                </div>
            </article>
        `;
    }).join('');
}

function upsertJob(job) {
    if (job.status === 'queued' || job.status === 'processing') {
        state.importPanelDismissed = false;
    }
    const idx = state.importJobs.findIndex(existing => existing.id === job.id);
    if (idx === -1) {
        state.importJobs.unshift(job);
    } else {
        state.importJobs[idx] = job;
    }
    state.importJobs = sortJobs(state.importJobs).slice(0, 10);
    renderImportJobs();
}

function onJobEvent(job) {
    const prev = state.importJobs.find(existing => existing.id === job.id);
    upsertJob(job);

    if (job.status === 'completed' && prev?.status !== 'completed') {
        showToast(`Import finished: ${job.indexed} indexed${job.skipped ? `, ${job.skipped} skipped` : ''}`, false);
        fetchCollections();
        if (job.collection === state.activeCollection) {
            refresh();
        }
    }

    if (job.status === 'failed' && prev?.status !== 'failed') {
        showToast(job.error || 'Import failed.', true);
        fetchCollections();
        if (job.collection === state.activeCollection) {
            refresh();
        }
    }
}

export async function fetchImportJobs() {
    if (!state.activeCollection) {
        state.importJobs = [];
        renderImportJobs();
        return;
    }
    try {
        const res = await fetch(`/api/imports${collectionParam(state.activeCollection)}`);
        if (!res.ok) return;
        const data = await res.json();
        state.importJobs = Array.isArray(data.jobs) ? data.jobs : [];
        if (state.importJobs.some(job => job.status === 'queued' || job.status === 'processing')) {
            state.importPanelDismissed = false;
        }
        renderImportJobs();
    } catch { }
}

export function closeImportStream() {
    if (state.importEventSource) {
        state.importEventSource.close();
        state.importEventSource = null;
    }
    state.importStreamCollection = null;
}

export function connectImportStream() {
    if (!state.activeCollection) {
        closeImportStream();
        state.importJobs = [];
        renderImportJobs();
        return;
    }
    if (state.importEventSource && state.importStreamCollection === state.activeCollection) {
        return;
    }

    closeImportStream();
    state.importStreamCollection = state.activeCollection;
    const url = `/api/imports/events${collectionParam(state.activeCollection)}`;
    const es = new EventSource(url);
    state.importEventSource = es;

    es.addEventListener('import', event => {
        try {
            const job = JSON.parse(event.data);
            onJobEvent(job);
        } catch { }
    });

    es.onerror = () => {
        closeImportStream();
        window.setTimeout(() => {
            if (state.activeCollection) {
                connectImportStream();
            }
        }, 2500);
    };
}

export async function queueImport(payload) {
    const res = await fetch(`/api/imports${collectionParam(state.activeCollection)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
    });
    if (!res.ok) {
        throw new Error(await res.text());
    }
    const data = await res.json();
    if (data.job) {
        upsertJob(data.job);
    }
    return data.job;
}

export function initImports() {
    dom.dismissImportPanelBtn.addEventListener('click', () => {
        state.importPanelDismissed = true;
        renderImportJobs();
    });
    renderImportJobs();
}
