/**
 * nms — Log Viewer JavaScript
 * Handles log searching, filtering, pagination, severity stats,
 * keyword highlighting, AI session integration, and export.
 */

'use strict';

let currentPage = 1;
const perPage = 50;

// ============================================
// Load Log Hosts for Filter Dropdown
// ============================================
async function loadHosts() {
    const hosts = await api('/api/logs/hosts');
    if (!hosts) return;

    const select = document.getElementById('log-host');
    hosts.forEach(host => {
        const opt = document.createElement('option');
        opt.value = host;
        opt.textContent = host;
        select.appendChild(opt);
    });
}

// ============================================
// Load Severity Stats Cards
// ============================================
async function loadLogStats() {
    const pp = getPeriodParams('log-period');
    const params = new URLSearchParams({ period: pp.period });
    if (pp.period === 'custom') {
        params.set('start', pp.start);
        params.set('end', pp.end);
    }
    const stats = await api(`/api/logs/stats?${params}`);
    if (!stats) return;

    const container = document.getElementById('severity-stats');
    const severityColors = {
        'emergency': { icon: '🔴', class: 'red' },
        'alert':     { icon: '🔴', class: 'red' },
        'critical':  { icon: '🔴', class: 'red' },
        'error':     { icon: '🟠', class: 'yellow' },
        'warning':   { icon: '🟡', class: 'yellow' },
        'notice':    { icon: '🔵', class: 'accent' },
        'info':      { icon: '🔵', class: 'accent' },
        'debug':     { icon: '⚪', class: 'accent' },
    };

    let html = '';
    stats.forEach(s => {
        const meta = severityColors[s.severity] || { icon: '⚪', class: 'accent' };
        html += `
            <div class="stat-card ${meta.class}" style="cursor:pointer"
                 onclick="filterBySeverity('${s.severity}')">
                <div class="stat-icon">${meta.icon}</div>
                <div class="stat-value">${s.count}</div>
                <div class="stat-label">${s.severity}</div>
            </div>
        `;
    });

    container.innerHTML = html;
}

function filterBySeverity(severity) {
    document.getElementById('log-severity').value = severity;
    currentPage = 1;
    searchLogs();
}

// ============================================
// Get current filter params (shared helper)
// ============================================
function getFilterParams() {
    const q = document.getElementById('log-search').value;
    const host = document.getElementById('log-host').value;
    const severity = document.getElementById('log-severity').value;
    const exact = document.getElementById('log-exact').checked;
    const periodParams = getPeriodParams('log-period');

    return { q, host, severity, exact, ...periodParams };
}

// ============================================
// Search Logs
// ============================================
async function searchLogs(resetPage = true) {
    if (resetPage) {
        currentPage = 1;
    }

    const filters = getFilterParams();

    const params = new URLSearchParams({
        q: filters.q,
        host: filters.host,
        severity: filters.severity,
        period: filters.period,
        page: currentPage,
        per_page: perPage,
    });
    if (filters.exact) {
        params.set('exact', 'true');
    }
    if (filters.period === 'custom') {
        params.set('start', filters.start);
        params.set('end', filters.end);
    }

    const data = await api(`/api/logs?${params}`);
    if (!data) return;

    renderLogs(data, filters.q);
}

// ============================================
// Render Log Table
// ============================================
function renderLogs(data, searchTerm) {
    const tbody = document.getElementById('log-table-body');
    const countEl = document.getElementById('log-count');

    if (!data.logs || data.logs.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="5">
                    <div class="empty-state">
                        <div class="empty-icon">📋</div>
                        <h3>Nenhum log encontrado</h3>
                        <p>Tente ajustar os filtros ou o período de busca.</p>
                    </div>
                </td>
            </tr>
        `;
        countEl.textContent = '0 logs';
        return;
    }

    countEl.textContent = `${data.total} logs`;

    const severityClasses = {
        'emergency': 'severity-critical',
        'alert': 'severity-critical',
        'critical': 'severity-critical',
        'error': 'severity-error',
        'warning': 'severity-warning',
        'notice': 'severity-info',
        'info': 'severity-info',
        'debug': 'severity-debug',
    };

    let html = '';
    data.logs.forEach(log => {
        const cls = severityClasses[log.severity] || 'severity-debug';
        const time = formatTime(log.time);
        // Escape HTML in message to prevent XSS
        const msg = escapeHtml(log.message);
        const app = escapeHtml(log.app_name || '');

        // Apply keyword highlight if searching
        const highlightedMsg = searchTerm ? highlightText(msg, searchTerm) : msg;

        html += `
            <tr>
                <td class="log-time">${time}</td>
                <td>${escapeHtml(log.host)}</td>
                <td><span class="badge ${cls}">${escapeHtml(log.severity)}</span></td>
                <td class="text-muted">${app}</td>
                <td class="log-message" title="${msg}">${highlightedMsg}</td>
            </tr>
        `;
    });

    tbody.innerHTML = html;

    // Pagination
    const pageInfo = document.getElementById('page-info');
    pageInfo.textContent = `Página ${data.page} — ${data.total} resultados`;

    document.getElementById('prev-page').disabled = data.page <= 1;
    document.getElementById('next-page').disabled = !data.has_more;
}

// ============================================
// Keyword Highlight
// ============================================
function highlightText(escapedHtml, term) {
    if (!term) return escapedHtml;

    // Escape the term for use in regex (the term is plain text, not HTML)
    const escapedTerm = escapeHtml(term).replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const regex = new RegExp(`(${escapedTerm})`, 'gi');

    return escapedHtml.replace(regex, '<span class="highlight">$1</span>');
}

// ============================================
// Pagination
// ============================================
function changePage(delta) {
    currentPage += delta;
    if (currentPage < 1) currentPage = 1;
    searchLogs(false);
}

// ============================================
// HTML Escaping (XSS protection)
// ============================================
function escapeHtml(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
}

// ============================================
// CSV Export
// ============================================
function exportCSV() {
    const filters = getFilterParams();
    const params = new URLSearchParams({
        q: filters.q,
        host: filters.host,
        severity: filters.severity,
        period: filters.period,
    });
    if (filters.exact) {
        params.set('exact', 'true');
    }
    if (filters.period === 'custom') {
        params.set('start', filters.start);
        params.set('end', filters.end);
    }
    window.location.href = `/api/logs/export?${params}`;
}

// ============================================
// TXT Export
// ============================================
function exportTXT() {
    const filters = getFilterParams();
    const params = new URLSearchParams({
        q: filters.q,
        host: filters.host,
        severity: filters.severity,
        period: filters.period,
    });
    if (filters.exact) {
        params.set('exact', 'true');
    }
    if (filters.period === 'custom') {
        params.set('start', filters.start);
        params.set('end', filters.end);
    }
    window.location.href = `/api/logs/export/txt?${params}`;
}

// ============================================
// Send to AI — Modal Management
// ============================================
let selectedAISessionId = null;

async function openSendToAI() {
    selectedAISessionId = null;
    document.getElementById('btn-confirm-send').disabled = true;

    const modal = document.getElementById('ai-session-modal');
    modal.classList.add('active');

    // Load sessions
    const sessions = await api('/api/ai/sessions');
    const container = document.getElementById('ai-modal-sessions');

    if (!sessions || sessions.length === 0) {
        container.innerHTML = `
            <div style="padding: 1rem; text-align: center; color: var(--text-muted); font-size: 0.8rem;">
                Nenhuma sessão encontrada. Clique em "+ Nova Sessão" para criar uma.
            </div>
        `;
        return;
    }

    let html = '';
    sessions.forEach(s => {
        const time = formatTimeShort(s.updated_at);
        html += `
            <div class="modal-session-item" data-id="${s.id}" onclick="selectAISession(${s.id}, this)">
                <span>${escapeHtml(s.title)}</span>
                <span class="modal-session-time">${time}</span>
            </div>
        `;
    });

    container.innerHTML = html;
}

function selectAISession(id, el) {
    selectedAISessionId = id;
    document.getElementById('btn-confirm-send').disabled = false;

    // Update visual selection
    document.querySelectorAll('#ai-modal-sessions .modal-session-item').forEach(item => {
        item.classList.remove('selected');
    });
    el.classList.add('selected');
}

function closeAIModal() {
    document.getElementById('ai-session-modal').classList.remove('active');
    selectedAISessionId = null;
}

async function createAndSendToAI() {
    // Create a new session and immediately send context to it
    const result = await api('/api/ai/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title: 'Logs ' + new Date().toLocaleDateString('pt-BR') }),
    });
    if (!result) return;

    selectedAISessionId = result.id;
    await confirmSendToAI();
}

async function confirmSendToAI() {
    if (!selectedAISessionId) return;

    const btn = document.getElementById('btn-confirm-send');
    btn.disabled = true;
    btn.textContent = '⏳ Enviando...';

    const filters = getFilterParams();

    const body = {
        host: filters.host,
        severity: filters.severity,
        period: filters.period,
        q: filters.q,
        exact: filters.exact,
        start: filters.start || '',
        end: filters.end || '',
    };

    const result = await api(`/api/ai/sessions/${selectedAISessionId}/context`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });

    btn.textContent = 'Enviar';
    btn.disabled = false;

    if (result) {
        closeAIModal();
        if (result.count > 0) {
            alert(`✅ ${result.count} logs adicionados ao contexto da sessão de IA.\n\nAcesse a página "Análise IA" para conversar sobre os logs.`);
        } else {
            alert('Nenhum log encontrado com os filtros atuais.');
        }
    }
}
