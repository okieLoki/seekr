import { dom } from './state.js';
import { logout } from './auth.js';

export function openSidebar() {
    dom.sidebar.classList.add('open');
    dom.sidebarOverlay.classList.add('show');
    document.body.style.overflow = 'hidden';
}

export function closeSidebar() {
    dom.sidebar.classList.remove('open');
    dom.sidebarOverlay.classList.remove('show');
    document.body.style.overflow = '';
}

export function initSidebar() {
    dom.menuBtn.addEventListener('click', openSidebar);
    dom.sidebarCloseBtn.addEventListener('click', closeSidebar);
    dom.sidebarOverlay.addEventListener('click', closeSidebar);

    const mq = window.matchMedia('(max-width: 900px)');
    mq.addEventListener('change', e => { if (!e.matches) closeSidebar(); });

    const logoutBtn = document.getElementById('logoutBtn');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', () => logout());
    }
}
