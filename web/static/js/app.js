/**
 * nms — Main Application JavaScript
 * Utilities and shared functionality
 */

'use strict';

// ============================================
// API Helper
// ============================================
async function api(url, options = {}) {
    try {
        const resp = await fetch(url, {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers,
            },
        });

        if (resp.status === 401) {
            window.location.href = '/login';
            return null;
        }

        if (!resp.ok) {
            console.error(`API error: ${resp.status} ${resp.statusText}`);
            return null;
        }

        return await resp.json();
    } catch (err) {
        console.error('API request failed:', err);
        return null;
    }
}

// ============================================
// Formatters
// ============================================
function formatBps(bps) {
    if (bps == null) return '—';
    if (bps >= 1e9) return (bps / 1e9).toFixed(2) + ' Gbps';
    if (bps >= 1e6) return (bps / 1e6).toFixed(2) + ' Mbps';
    if (bps >= 1e3) return (bps / 1e3).toFixed(2) + ' Kbps';
    return bps.toFixed(0) + ' bps';
}

function formatTime(isoStr) {
    const d = new Date(isoStr);
    return d.toLocaleString('pt-BR', {
        day: '2-digit', month: '2-digit',
        hour: '2-digit', minute: '2-digit', second: '2-digit',
    });
}

function formatTimeShort(isoStr) {
    const d = new Date(isoStr);
    return d.toLocaleTimeString('pt-BR', {
        hour: '2-digit', minute: '2-digit',
    });
}

// ============================================
// Sidebar mobile toggle
// ============================================
document.addEventListener('DOMContentLoaded', function() {
    // Close modals on overlay click
    document.querySelectorAll('.modal-overlay').forEach(overlay => {
        overlay.addEventListener('click', function(e) {
            if (e.target === this) {
                this.classList.remove('active');
            }
        });
    });

    // Close modals on Escape
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape') {
            document.querySelectorAll('.modal-overlay.active').forEach(m => {
                m.classList.remove('active');
            });
        }
    });

    // Start polling alarms
    fetchAlarms();
    setInterval(fetchAlarms, 30000); // 30s
});

// ============================================
// Alarms
// ============================================
async function fetchAlarms() {
    const alarms = await api('/api/alarms');
    if (!alarms) return;

    const navItem = document.getElementById('nav-alarms');
    const badge = document.getElementById('alarm-count');
    
    if (alarms.length > 0) {
        navItem.style.display = 'flex';
        badge.textContent = alarms.length;
    } else {
        navItem.style.display = 'none';
        badge.textContent = '0';
    }

    // Se o modal estiver aberto, atualizar a tabela
    const modal = document.getElementById('alarms-modal');
    if (modal && modal.classList.contains('active')) {
        renderAlarmsTable(alarms);
    }
}

async function showAlarmsModal() {
    const modal = document.getElementById('alarms-modal');
    if (modal) modal.classList.add('active');
    
    const alarms = await api('/api/alarms');
    if (alarms) renderAlarmsTable(alarms);
}

function renderAlarmsTable(alarms) {
    const tbody = document.getElementById('alarms-table-body');
    if (!tbody) return;

    if (alarms.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;">Nenhum alarme ativo.</td></tr>';
        return;
    }

    tbody.innerHTML = alarms.map(a => `
        <tr>
            <td>${formatTimeShort(a.created_at)}</td>
            <td>Dev ${a.device_id}</td>
            <td><span class="badge badge-down">${a.severity.toUpperCase()}</span></td>
            <td>${a.message}</td>
            <td>
                <button class="btn btn-sm" onclick="resolveAlarm(${a.id})">✅ Resolver</button>
            </td>
        </tr>
    `).join('');
}

async function resolveAlarm(id) {
    const res = await api(`/api/alarms/${id}/resolve`, { method: 'POST' });
    if (res && res.status === 'ok') {
        fetchAlarms();
    }
}
