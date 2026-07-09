/**
 * nms — Log Viewer JavaScript
 * Handles log searching, filtering, pagination, and severity stats.
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
    const period = document.getElementById('log-period').value;
    const stats = await api(`/api/logs/stats?period=${period}`);
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
// Search Logs
// ============================================
async function searchLogs(resetPage = true) {
    if (resetPage) {
        currentPage = 1;
    }

    const q = document.getElementById('log-search').value;
    const host = document.getElementById('log-host').value;
    const severity = document.getElementById('log-severity').value;
    const period = document.getElementById('log-period').value;

    const params = new URLSearchParams({
        q, host, severity, period,
        page: currentPage,
        per_page: perPage,
    });

    const data = await api(`/api/logs?${params}`);
    if (!data) return;

    renderLogs(data);
}

// ============================================
// Render Log Table
// ============================================
function renderLogs(data) {
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

        html += `
            <tr>
                <td class="log-time">${time}</td>
                <td>${escapeHtml(log.host)}</td>
                <td><span class="badge ${cls}">${escapeHtml(log.severity)}</span></td>
                <td class="text-muted">${app}</td>
                <td class="log-message" title="${msg}">${msg}</td>
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
    const q = document.getElementById('log-search').value;
    const host = document.getElementById('log-host').value;
    const severity = document.getElementById('log-severity').value;
    const period = document.getElementById('log-period').value;

    const params = new URLSearchParams({ q, host, severity, period });
    window.location.href = `/api/logs/export?${params}`;
}

