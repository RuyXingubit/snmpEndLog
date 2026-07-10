/**
 * nms — Chart.js configuration and helpers
 */

'use strict';

// ============================================
// Chart.js Global Defaults
// ============================================
if (typeof Chart !== 'undefined') {
    // Utilize CSS variables defined in style.css
    Chart.defaults.color = '#94A3B8'; // --text-secondary
    Chart.defaults.borderColor = '#323842'; // --border-subtle
    Chart.defaults.font.family = "'JetBrains Mono', monospace";
    Chart.defaults.font.size = 11;
    Chart.defaults.plugins.legend.labels.usePointStyle = true;
    Chart.defaults.plugins.legend.labels.pointStyleWidth = 8;
    Chart.defaults.animation.duration = 0; // Brutalist: no animation or very fast
}

// Store chart instances for cleanup
const chartInstances = {};

function destroyChart(id) {
    if (chartInstances[id]) {
        chartInstances[id].destroy();
        delete chartInstances[id];
    }
}

// ============================================
// Create a line chart
// ============================================
function createLineChart(canvasId, labels, datasets, options = {}) {
    destroyChart(canvasId);

    const canvas = document.getElementById(canvasId);
    if (!canvas) return null;

    const ctx = canvas.getContext('2d');

    const chart = new Chart(ctx, {
        type: 'line',
        data: { labels, datasets },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            interaction: {
                mode: 'index',
                intersect: false,
            },
            plugins: {
                legend: {
                    position: 'top',
                    align: 'end',
                },
                tooltip: {
                    backgroundColor: '#1D2127', // --bg-surface
                    borderColor: '#4A5568', // --border-strong
                    borderWidth: 1,
                    titleFont: { weight: '600' },
                    padding: 8,
                    cornerRadius: 2, // Sharp corners
                    callbacks: {
                        label: function(context) {
                            let label = context.dataset.label || '';
                            if (label) {
                                label += ': ';
                            }
                            if (context.parsed.y !== null) {
                                label += options.yFormat ? options.yFormat(context.parsed.y) : context.parsed.y;
                            }
                            return label;
                        }
                    }
                },
            },
            scales: {
                x: {
                    grid: { display: false },
                    ticks: {
                        maxTicksLimit: 8,
                        callback: function(val) {
                            return formatTimeShort(this.getLabelForValue(val));
                        },
                    },
                },
                y: {
                    beginAtZero: true,
                    grid: {
                        color: '#323842', // solid border subtle
                    },
                    ticks: {
                        callback: options.yFormat || (v => v),
                    },
                },
            },
            elements: {
                point: { radius: 0, hitRadius: 10, hoverRadius: 0 },
                line: { tension: 0, borderWidth: 2 }, // Brutalist: no curve (stepped/straight lines)
            },
        },
    });

    chartInstances[canvasId] = chart;
    return chart;
}

// ============================================
// Device Detail Charts
// ============================================
async function initDeviceCharts(deviceId, period) {
    // CPU & Memory chart
    const sysData = await api(`/api/metrics/system?device_id=${deviceId}&period=${period}`);
    if (sysData) {
        const labels = (sysData.cpu || []).map(p => p.time);

        createLineChart('chart-cpu', labels, [
            {
                label: 'CPU %',
                data: (sysData.cpu || []).map(p => p.value),
                borderColor: '#3B82F6', // Blue
                backgroundColor: 'transparent',
                fill: false,
            },
        ], {
            yFormat: v => v + '%',
        });

        const memLabels = (sysData.memory || []).map(p => p.time);
        createLineChart('chart-memory', memLabels, [
            {
                label: 'Memória %',
                data: (sysData.memory || []).map(p => p.value),
                borderColor: '#F59E0B', // Orange
                backgroundColor: 'transparent',
                fill: false,
            },
        ], {
            yFormat: v => v + '%',
        });

        // Temperature chart (if available)
        if (sysData.temperature && sysData.temperature.length > 0) {
            const tChart = document.getElementById('temperature-chart-card');
            if (tChart) tChart.style.display = 'block';

            const tempLabels = sysData.temperature.map(p => p.time);
            createLineChart('chart-temperature', tempLabels, [
                {
                    label: 'Temperatura °C',
                    data: sysData.temperature.map(p => p.value),
                    borderColor: '#EF4444', // Red
                    backgroundColor: 'transparent',
                    fill: false,
                },
            ], {
                yFormat: v => v + ' °C',
            });
        }

        // PPPoE chart (if available)
        if (sysData.pppoe && sysData.pppoe.length > 0) {
            const pCard = document.getElementById('pppoe-card');
            const pChart = document.getElementById('pppoe-chart-card');
            if (pCard) pCard.style.display = 'block';
            if (pChart) pChart.style.display = 'block';
            
            // Set current online users in card
            const latestPPPoE = sysData.pppoe[sysData.pppoe.length - 1].value;
            const pVal = document.getElementById('pppoe-value');
            if (pVal) pVal.textContent = latestPPPoE;

            const pppoeLabels = sysData.pppoe.map(p => p.time);
            createLineChart('chart-pppoe', pppoeLabels, [
                {
                    label: 'PPPoE Online',
                    data: sysData.pppoe.map(p => p.value),
                    borderColor: '#10B981', // Green
                    backgroundColor: 'transparent',
                    fill: false,
                    stepped: true,
                },
            ]);
        } else {
            const pCard = document.getElementById('pppoe-card');
            const pChart = document.getElementById('pppoe-chart-card');
            if (pCard) pCard.style.display = 'none';
            if (pChart) pChart.style.display = 'none';
        }
    }

    // Ping chart
    const pingData = await api(`/api/metrics/ping?device_id=${deviceId}&period=${period}`);
    if (pingData) {
        const labels = (pingData.rtt || []).map(p => p.time);
        createLineChart('chart-ping', labels, [
            {
                label: 'RTT (ms)',
                data: (pingData.rtt || []).map(p => p.value),
                borderColor: '#10B981', // Green
                backgroundColor: 'transparent',
                fill: false,
            },
            {
                label: 'Packet Loss %',
                data: (pingData.packet_loss || []).map(p => p.value),
                borderColor: '#EF4444', // Red
                backgroundColor: 'transparent',
                fill: false,
                yAxisID: 'y1',
            },
        ]);
    }
}

// ============================================
// Interface Traffic Chart (Modal)
// ============================================
async function showTrafficChart(deviceId, ifIndex, ifDescr) {
    const titleEl = document.getElementById('traffic-modal-title');
    if (titleEl) titleEl.textContent = `Tráfego — ${ifDescr}`;
    
    const modal = document.getElementById('traffic-modal');
    if (modal) modal.classList.add('active');

    // Get active period
    const activeBtn = document.querySelector('#period-selector .chart-period.active');
    let period = activeBtn ? activeBtn.dataset.period : '1h';
    if (period === 'custom') {
        const s = document.getElementById('device-period-start').value;
        const e = document.getElementById('device-period-end').value;
        period = `custom&start=${s}&end=${e}`;
    }

    const data = await api(`/api/metrics/traffic?device_id=${deviceId}&if_index=${ifIndex}&period=${period}`);
    if (data) {
        const labels = (data.in_bps || []).map(p => p.time);
        createLineChart('chart-traffic-detail', labels, [
            {
                label: '↓ In',
                data: (data.in_bps || []).map(p => p.value),
                borderColor: '#10B981', // Green
                backgroundColor: 'transparent',
                fill: false,
            },
            {
                label: '↑ Out',
                data: (data.out_bps || []).map(p => p.value),
                borderColor: '#3B82F6', // Blue
                backgroundColor: 'transparent',
                fill: false,
            },
        ], {
            yFormat: v => formatBps(v),
        });
    }
}

// ============================================
// BGP State Chart (Modal)
// ============================================
async function showBgpChart(deviceId, peerAddr) {
    const titleEl = document.getElementById('traffic-modal-title');
    if (titleEl) titleEl.textContent = `BGP — ${peerAddr}`;
    
    const modal = document.getElementById('traffic-modal');
    if (modal) modal.classList.add('active');

    // Get active period
    const activeBtn = document.querySelector('#period-selector .chart-period.active');
    let period = activeBtn ? activeBtn.dataset.period : '1h';
    if (period === 'custom') {
        const s = document.getElementById('device-period-start').value;
        const e = document.getElementById('device-period-end').value;
        period = `custom&start=${s}&end=${e}`;
    }

    const data = await api(`/api/metrics/bgp?device_id=${deviceId}&peer_addr=${peerAddr}&period=${period}`);
    if (data) {
        const labels = (data.state || []).map(p => p.time);
        createLineChart('chart-traffic-detail', labels, [
            {
                label: 'Status (1=UP, 0=DOWN)',
                data: (data.state || []).map(p => p.value),
                borderColor: '#10B981', // Green
                backgroundColor: 'transparent',
                fill: false,
                stepped: true, // Crucial para gráfico de estado (degraus)
            },
        ], {
            yFormat: v => (v === 1 ? 'UP' : (v === 0 ? 'DOWN' : v)),
        });
    }
}
