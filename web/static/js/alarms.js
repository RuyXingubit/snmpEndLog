let alarmsRefreshTimer;
let currentAlarmsData = [];
let knownDevices = new Map();

document.addEventListener('DOMContentLoaded', () => {
    // Restore saved view mode preference
    const savedMode = localStorage.getItem('alarms_view_mode');
    const viewSelect = document.getElementById('alarm-view-mode');
    if (savedMode && viewSelect) {
        viewSelect.value = savedMode;
    }

    fetchAlarmsPage();
    // Auto-refresh every 30 seconds
    alarmsRefreshTimer = setInterval(fetchAlarmsPage, 30000);
});

function changeViewMode() {
    const mode = document.getElementById('alarm-view-mode').value;
    localStorage.setItem('alarms_view_mode', mode);
    renderCurrentAlarms();
}

function updateDeviceDropdown(alarms) {
    const select = document.getElementById('alarm-device');
    if (!select) return;

    const currentSelected = select.value;

    // Accumulate devices into knownDevices map
    alarms.forEach(a => {
        if (a.device_id && !knownDevices.has(a.device_id)) {
            knownDevices.set(a.device_id, a.device_name || `Dispositivo ${a.device_id}`);
        }
    });

    // Rebuild options while preserving selection
    let html = '<option value="">Todos os Equipamentos</option>';
    const sortedDevices = Array.from(knownDevices.entries()).sort((a, b) => a[1].localeCompare(b[1]));
    sortedDevices.forEach(([id, name]) => {
        const selected = String(id) === String(currentSelected) ? 'selected' : '';
        html += `<option value="${id}" ${selected}>${escapeHtml(name)}</option>`;
    });

    select.innerHTML = html;
}

async function fetchAlarmsPage() {
    const statusEl = document.getElementById('alarm-status');
    const deviceEl = document.getElementById('alarm-device');
    const status = statusEl ? statusEl.value : 'active';
    const deviceId = deviceEl ? deviceEl.value : '';
    const countSpan = document.getElementById('alarm-count');

    let url = `/api/alarms?status=${encodeURIComponent(status)}`;
    if (deviceId) {
        url += `&device_id=${encodeURIComponent(deviceId)}`;
    }

    try {
        const data = await api(url);
        if (data) {
            currentAlarmsData = data;
            updateDeviceDropdown(data);
            renderCurrentAlarms();
        }
    } catch (err) {
        console.error('Erro ao buscar alarmes:', err);
        const tbody = document.getElementById('alarms-page-table-body');
        if (tbody) {
            tbody.innerHTML = `<tr><td colspan="8" class="empty-state text-danger">Erro ao carregar alarmes.</td></tr>`;
        }
    }
}

function applyClientFilter() {
    renderCurrentAlarms();
}

function getFilteredAlarms() {
    const searchInput = document.getElementById('alarm-search');
    const term = searchInput ? searchInput.value.trim().toLowerCase() : '';

    if (!term) return currentAlarmsData;

    return currentAlarmsData.filter(a => {
        const devName = (a.device_name || '').toLowerCase();
        const msg = (a.message || '').toLowerCase();
        const name = (a.name || '').toLowerCase();
        const entityId = (a.entity_id || '').toLowerCase();
        const severity = (a.severity || '').toLowerCase();
        return devName.includes(term) || msg.includes(term) || name.includes(term) || entityId.includes(term) || severity.includes(term);
    });
}

function formatPortInfo(a) {
    if (a.entity_type === 'interface') {
        const match = a.name.match(/Interface (.+) Down/);
        if (match && match[1]) {
            return `<strong>${escapeHtml(match[1])}</strong> <span class="text-muted">(ID: ${escapeHtml(a.entity_id)})</span>`;
        }
        return `<strong>${escapeHtml(a.name)}</strong> <span class="text-muted">(ID: ${escapeHtml(a.entity_id)})</span>`;
    } else if (a.entity_type === 'bgp_peer') {
        return `<strong>${escapeHtml(a.entity_id)}</strong> <span class="text-muted">(BGP Peer)</span>`;
    }
    return `<strong>${escapeHtml(a.entity_id)}</strong>`;
}

function renderCurrentAlarms() {
    const alarms = getFilteredAlarms();
    const countSpan = document.getElementById('alarm-count');
    if (countSpan) {
        countSpan.textContent = `${alarms.length} alarme${alarms.length === 1 ? '' : 's'} encontrado${alarms.length === 1 ? '' : 's'}`;
    }

    const mode = document.getElementById('alarm-view-mode') ? document.getElementById('alarm-view-mode').value : 'flat';
    const flatContainer = document.getElementById('alarms-flat-container');
    const groupedContainer = document.getElementById('alarms-grouped-container');

    if (mode === 'grouped') {
        if (flatContainer) flatContainer.style.display = 'none';
        if (groupedContainer) groupedContainer.style.display = 'block';
        renderGroupedView(alarms);
    } else {
        if (groupedContainer) groupedContainer.style.display = 'none';
        if (flatContainer) flatContainer.style.display = 'block';
        renderFlatView(alarms);
    }
}

function renderFlatView(alarms) {
    const tbody = document.getElementById('alarms-page-table-body');
    if (!tbody) return;

    if (alarms.length === 0) {
        tbody.innerHTML = `<tr><td colspan="8" class="empty-state">Nenhum alarme encontrado com os filtros selecionados.</td></tr>`;
        return;
    }

    tbody.innerHTML = alarms.map(a => {
        const isResolved = a.status === 'resolved';
        const statusBadge = isResolved 
            ? `<span class="badge" style="background: var(--status-up); color: white;">RESOLVIDO</span>`
            : `<span class="badge badge-down">ATIVO</span>`;
        
        let actions = '';
        if (!isResolved) {
            actions = `<button class="btn btn-sm" onclick="resolveAlarmPage(${a.id})">✅ Resolver</button>`;
        }

        const portInfo = formatPortInfo(a);
        const deviceDisplay = a.device_id 
            ? `<a href="/devices/${a.device_id}" style="font-weight: 600;">${escapeHtml(a.device_name || `Dev ${a.device_id}`)}</a>`
            : `<strong>${escapeHtml(a.device_name || `Dev ${a.device_id}`)}</strong>`;

        return `
        <tr class="${isResolved ? 'text-muted' : ''}">
            <td class="log-time" style="white-space: nowrap;">${formatDateTime(a.created_at)}</td>
            <td>${deviceDisplay}</td>
            <td>${portInfo}</td>
            <td><span class="badge ${a.severity === 'critical' ? 'badge-down' : 'badge-warning'}">${escapeHtml(a.severity.toUpperCase())}</span></td>
            <td>${statusBadge}</td>
            <td class="log-message" style="white-space: normal;">${escapeHtml(a.message)}</td>
            <td class="log-time" style="white-space: nowrap;">${isResolved && a.resolved_at ? formatDateTime(a.resolved_at) : '-'}</td>
            <td>${actions}</td>
        </tr>
        `;
    }).join('');
}

function renderGroupedView(alarms) {
    const container = document.getElementById('alarms-grouped-container');
    if (!container) return;

    if (alarms.length === 0) {
        container.innerHTML = `
        <div class="card">
            <div class="empty-state">Nenhum alarme encontrado com os filtros selecionados.</div>
        </div>`;
        return;
    }

    // Group alarms by device_id
    const groups = new Map();
    alarms.forEach(a => {
        const key = a.device_id || 0;
        if (!groups.has(key)) {
            groups.set(key, {
                deviceId: a.device_id,
                deviceName: a.device_name || (a.device_id ? `Dispositivo ${a.device_id}` : 'Desconhecido'),
                alarms: []
            });
        }
        groups.get(key).alarms.push(a);
    });

    let html = '';
    groups.forEach(group => {
        const activeCount = group.alarms.filter(a => a.status !== 'resolved').length;
        const resolvedCount = group.alarms.filter(a => a.status === 'resolved').length;

        const rows = group.alarms.map(a => {
            const isResolved = a.status === 'resolved';
            const statusBadge = isResolved 
                ? `<span class="badge" style="background: var(--status-up); color: white;">RESOLVIDO</span>`
                : `<span class="badge badge-down">ATIVO</span>`;
            
            let actions = '';
            if (!isResolved) {
                actions = `<button class="btn btn-sm" onclick="resolveAlarmPage(${a.id})">✅ Resolver</button>`;
            }

            const portInfo = formatPortInfo(a);

            return `
            <tr class="${isResolved ? 'text-muted' : ''}">
                <td class="log-time" style="white-space: nowrap;">${formatDateTime(a.created_at)}</td>
                <td>${portInfo}</td>
                <td><span class="badge ${a.severity === 'critical' ? 'badge-down' : 'badge-warning'}">${escapeHtml(a.severity.toUpperCase())}</span></td>
                <td>${statusBadge}</td>
                <td class="log-message" style="white-space: normal;">${escapeHtml(a.message)}</td>
                <td class="log-time" style="white-space: nowrap;">${isResolved && a.resolved_at ? formatDateTime(a.resolved_at) : '-'}</td>
                <td>${actions}</td>
            </tr>
            `;
        }).join('');

        const deviceLink = group.deviceId 
            ? `<a href="/devices/${group.deviceId}" class="btn btn-sm" style="margin-left: auto; text-decoration: none;">Ver Detalhes →</a>`
            : '';

        html += `
        <div class="card mb-md">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--space-md); padding-bottom: var(--space-sm); border-bottom: 1px solid var(--border-subtle); flex-wrap: wrap; gap: var(--space-sm);">
                <div style="display: flex; align-items: center; gap: var(--space-sm);">
                    <span style="font-size: 1.25rem;">🖥️</span>
                    <strong style="font-size: 1.05rem; color: var(--text-primary);">${escapeHtml(group.deviceName)}</strong>
                    ${group.deviceId ? `<span class="text-muted" style="font-size: 0.8rem;">(ID: ${group.deviceId})</span>` : ''}
                </div>
                <div style="display: flex; align-items: center; gap: var(--space-sm);">
                    ${activeCount > 0 ? `<span class="badge badge-down">${activeCount} ATIVO${activeCount > 1 ? 'S' : ''}</span>` : ''}
                    ${resolvedCount > 0 ? `<span class="badge" style="background: var(--status-up); color: white;">${resolvedCount} RESOLVIDO${resolvedCount > 1 ? 'S' : ''}</span>` : ''}
                    ${deviceLink}
                </div>
            </div>
            <div class="table-container">
                <table>
                    <thead>
                        <tr>
                            <th>Data / Hora</th>
                            <th>Porta / Interface</th>
                            <th>Severidade</th>
                            <th>Status</th>
                            <th>Mensagem</th>
                            <th>Resolvido Em</th>
                            <th>Ações</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${rows}
                    </tbody>
                </table>
            </div>
        </div>
        `;
    });

    container.innerHTML = html;
}

async function resolveAlarmPage(id) {
    const res = await api(`/api/alarms/${id}/resolve`, { method: 'POST' });
    if (res && res.status === 'ok') {
        fetchAlarmsPage();
        // Also update the global sidebar alarms
        if (typeof fetchAlarms === 'function') {
            fetchAlarms();
        }
    }
}

