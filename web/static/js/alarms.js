let alarmsRefreshTimer;

document.addEventListener('DOMContentLoaded', () => {
    fetchAlarmsPage();
    // Auto-refresh every 30 seconds
    alarmsRefreshTimer = setInterval(fetchAlarmsPage, 30000);
});

async function fetchAlarmsPage() {
    const status = document.getElementById('alarm-status').value;
    const tbody = document.getElementById('alarms-page-table-body');
    const countSpan = document.getElementById('alarm-count');

    try {
        const data = await api(`/api/alarms?status=${status}`);
        
        if (data) {
            countSpan.textContent = `${data.length} alarmes encontrados`;
            
            if (data.length === 0) {
                tbody.innerHTML = `<tr><td colspan="8" class="empty-state">Nenhum alarme encontrado com o status selecionado.</td></tr>`;
                return;
            }

            tbody.innerHTML = data.map(a => {
                const isResolved = a.status === 'resolved';
                const statusBadge = isResolved 
                    ? `<span class="badge" style="background: var(--status-up); color: white;">RESOLVIDO</span>`
                    : `<span class="badge badge-down">ATIVO</span>`;
                
                let actions = '';
                if (!isResolved) {
                    actions = `<button class="btn btn-sm" onclick="resolveAlarmPage(${a.id})">✅ Resolver</button>`;
                }
                
                // Extrair nome da porta (entity_id ou nome descritivo)
                // name format is typically: "Interface [if_name] Down"
                // Let's extract the interface name from the string if possible, or just show entity_id + name
                let portInfo = a.entity_id;
                if (a.entity_type === 'interface') {
                    // Try to match "Interface X Down"
                    const match = a.name.match(/Interface (.+) Down/);
                    if (match && match[1]) {
                        portInfo = `<strong>${match[1]}</strong> (ID: ${a.entity_id})`;
                    } else {
                        portInfo = `<strong>${a.name}</strong> (ID: ${a.entity_id})`;
                    }
                } else if (a.entity_type === 'bgp_peer') {
                    portInfo = `<strong>${a.entity_id}</strong>`;
                }

                return `
                <tr class="${isResolved ? 'text-muted' : ''}">
                    <td>${formatTimeShort(a.created_at)}</td>
                    <td><strong>${a.device_name || `Dev ${a.device_id}`}</strong></td>
                    <td>${portInfo}</td>
                    <td><span class="badge ${a.severity === 'critical' ? 'badge-down' : 'badge-warning'}">${a.severity.toUpperCase()}</span></td>
                    <td>${statusBadge}</td>
                    <td>${a.message}</td>
                    <td>${isResolved && a.resolved_at ? formatTimeShort(a.resolved_at) : '-'}</td>
                    <td>${actions}</td>
                </tr>
                `;
            }).join('');
        }
    } catch (err) {
        tbody.innerHTML = `<tr><td colspan="8" class="empty-state text-danger">Erro ao carregar alarmes.</td></tr>`;
    }
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
