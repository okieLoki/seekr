export let isAuthenticated = false;

export async function checkAuth() {
    try {
        const res = await fetch('/api/stats', { method: 'GET' });
        if (res.status === 401) {
            isAuthenticated = false;
            return false;
        }
        isAuthenticated = true;
        return true;
    } catch {
        isAuthenticated = false;
        return false;
    }
}

export async function login(username, password) {
    try {
        const res = await fetch('/api/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password }),
        });
        const data = await res.json();
        if (res.ok) {
            isAuthenticated = true;
            return { ok: true };
        }
        return { ok: false, error: data.error || 'Login failed' };
    } catch {
        return { ok: false, error: 'Network error. Is the server running?' };
    }
}

export async function logout() {
    await fetch('/api/logout', { method: 'POST' });
    isAuthenticated = false;
    window._teardownApp && window._teardownApp();
    showLoginScreen();
}

let loginScreen = null;
let appRoot = null;

export function initAuth(appRootEl) {
    appRoot = appRootEl;
    _buildLoginScreen();
}

function _buildLoginScreen() {
    loginScreen = document.getElementById('loginScreen');
}

export function showLoginScreen() {
    window._teardownApp && window._teardownApp();
    if (loginScreen) loginScreen.classList.remove('hidden');
    if (appRoot) appRoot.classList.add('hidden');
    // Clear password field
    const pwd = document.getElementById('loginPassword');
    if (pwd) pwd.value = '';
    const err = document.getElementById('loginError');
    if (err) err.textContent = '';
}

export function hideLoginScreen() {
    if (loginScreen) loginScreen.classList.add('hidden');
    if (appRoot) appRoot.classList.remove('hidden');
}

export function setupLoginForm() {
    const form = document.getElementById('loginForm');
    const btn = document.getElementById('loginBtn');
    const errorEl = document.getElementById('loginError');

    if (!form) return;

    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('loginUsername').value.trim();
        const password = document.getElementById('loginPassword').value;

        if (!username || !password) {
            errorEl.textContent = 'Please enter username and password.';
            return;
        }

        btn.disabled = true;
        btn.textContent = 'Signing in…';
        errorEl.textContent = '';

        const result = await login(username, password);
        btn.disabled = false;
        btn.textContent = 'Sign in';

        if (result.ok) {
            hideLoginScreen();
            window._initApp && window._initApp();
        } else {
            errorEl.textContent = result.error;
            form.classList.add('shake');
            setTimeout(() => form.classList.remove('shake'), 400);
        }
    });
}
