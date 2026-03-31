import { state, dom } from './state.js';

export function updateBoostBadge() {
    const active = state.boostRows.filter(r => r.field.trim() && r.weight > 0).length;
    dom.boostBadge.textContent = active;
    dom.boostBadge.style.display = active > 0 ? 'flex' : 'none';
    dom.boostToggleBtn.classList.toggle('has-boosts', active > 0);
}

export function renderBoostRows() {
    dom.boostRowsCont.innerHTML = '';
    state.boostRows.forEach((row, i) => {
        const el = document.createElement('div');
        el.className = 'boost-row';
        el.innerHTML = `
            <input class="boost-field-input" list="fieldSuggestions" type="text"
                placeholder="Field name (e.g. Title)" value="${row.field}" autocomplete="off">
            <span class="boost-x-label">×</span>
            <input class="boost-weight-input" type="number" min="1" max="20" step="0.5" value="${row.weight}">
            <button class="boost-del-btn" title="Remove">×</button>`;
        el.querySelector('.boost-field-input').addEventListener('input', e => {
            state.boostRows[i].field = e.target.value;
            updateBoostBadge();
            if (state.isSearch) window._triggerSearch();
        });
        el.querySelector('.boost-weight-input').addEventListener('input', e => {
            state.boostRows[i].weight = parseFloat(e.target.value) || 1;
            updateBoostBadge();
            if (state.isSearch) window._triggerSearch();
        });
        el.querySelector('.boost-del-btn').addEventListener('click', () => {
            state.boostRows.splice(i, 1);
            renderBoostRows();
            updateBoostBadge();
            if (state.isSearch) window._triggerSearch();
        });
        dom.boostRowsCont.appendChild(el);
    });
}

export function getBoosts() {
    const m = {};
    state.boostRows.forEach(r => { if (r.field.trim() && r.weight > 0) m[r.field.trim()] = r.weight; });
    return Object.keys(m).length ? m : undefined;
}

export function initBoost() {
    dom.boostToggleBtn.addEventListener('click', () => {
        const open = dom.boostDrawer.style.display !== 'none';
        dom.boostDrawer.style.display = open ? 'none' : 'block';
        dom.boostToggleBtn.classList.toggle('active', !open);
    });

    dom.addBoostRowBtn.addEventListener('click', () => {
        state.boostRows.push({ field: '', weight: 2 });
        renderBoostRows();
        dom.boostRowsCont.lastChild?.querySelector('.boost-field-input')?.focus();
        updateBoostBadge();
    });
}
