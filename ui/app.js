import { initSidebar } from './sidebar.js';
import { initCollections, fetchCollections } from './collections.js';
import { initBoost } from './boost.js';
import { initDocs, fetchStats, fetchDocs, refresh, triggerSearch } from './docs.js';
import { initModals } from './modals.js';
import { initAuth, checkAuth, showLoginScreen, hideLoginScreen, setupLoginForm } from './auth.js';

window._refresh = refresh;
window._triggerSearch = triggerSearch;

function initApp() {
    initSidebar();
    initCollections();
    initBoost();
    initDocs();
    initModals();

    fetchCollections();
    fetchStats();
    fetchDocs(1);
}

window._initApp = initApp;

document.addEventListener('DOMContentLoaded', async () => {
    const appRoot = document.getElementById('appRoot');
    initAuth(appRoot);
    setupLoginForm();

    const authed = await checkAuth();
    if (authed) {
        hideLoginScreen();
        initApp();
    } else {
        showLoginScreen();
    }
});