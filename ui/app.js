import { initSidebar } from './sidebar.js';
import { initCollections, fetchCollections } from './collections.js';
import { initBoost } from './boost.js';
import { initDocs, fetchStats, fetchDocs, refresh, triggerSearch } from './docs.js';
import { initModals } from './modals.js';
import { initAuth, checkAuth, showLoginScreen, hideLoginScreen, setupLoginForm } from './auth.js';
import { initImports, fetchImportJobs, connectImportStream, closeImportStream } from './imports.js';

window._refresh = refresh;
window._triggerSearch = triggerSearch;
let appInitialized = false;

window._onCollectionChange = async () => {
    connectImportStream();
    await fetchImportJobs();
};

function initApp() {
    if (!appInitialized) {
        initSidebar();
        initCollections();
        initBoost();
        initDocs();
        initImports();
        initModals();
        appInitialized = true;
    }

    fetchCollections();
    fetchStats();
    fetchDocs(1);
    connectImportStream();
    fetchImportJobs();
}

window._initApp = initApp;
window._teardownApp = () => {
    closeImportStream();
};

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
