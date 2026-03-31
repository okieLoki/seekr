import { initSidebar } from './sidebar.js';
import { initCollections, fetchCollections } from './collections.js';
import { initBoost } from './boost.js';
import { initDocs, fetchStats, fetchDocs, refresh, triggerSearch } from './docs.js';
import { initModals } from './modals.js';

window._refresh = refresh;
window._triggerSearch = triggerSearch;

document.addEventListener('DOMContentLoaded', () => {
    initSidebar();
    initCollections();
    initBoost();
    initDocs();
    initModals();

    fetchCollections();
    fetchStats();
    fetchDocs(1);
});