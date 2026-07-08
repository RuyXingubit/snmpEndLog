/**
 * snmpEndLog — Main Application JavaScript
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
});
