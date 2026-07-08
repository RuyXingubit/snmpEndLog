/**
 * snmpEndLog — Chart.js configuration and helpers
 */

'use strict';

// ============================================
// Chart.js Global Defaults
// ============================================
if (typeof Chart !== 'undefined') {
    Chart.defaults.color = '#94a3b8';
    Chart.defaults.borderColor = 'rgba(56, 189, 248, 0.1)';
    Chart.defaults.font.family = "'Inter', sans-serif";
    Chart.defaults.font.size = 11;
    Chart.defaults.plugins.legend.labels.usePointStyle = true;
    Chart.defaults.plugins.legend.labels.pointStyleWidth = 8;
    Chart.defaults.animation.duration = 500;
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
                    backgroundColor: 'rgba(17, 24, 39, 0.95)',
                    borderColor: 'rgba(56, 189, 248, 0.2)',
                    borderWidth: 1,
                    titleFont: { weight: '600' },
                    padding: 10,
                    cornerRadius: 8,
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
                        color: 'rgba(56, 189, 248, 0.05)',
                    },
                    ticks: {
                        callback: options.yFormat || (v => v),
                    },
                },
            },
            elements: {
                point: { radius: 0, hitRadius: 10, hoverRadius: 4 },
                line: { tension: 0.3, borderWidth: 2 },
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
                borderColor: '#38bdf8',
                backgroundColor: 'rgba(56, 189, 248, 0.1)',
                fill: true,
            },
        ], {
            yFormat: v => v + '%',
        });

        const memLabels = (sysData.memory || []).map(p => p.time);
        createLineChart('chart-memory', memLabels, [
            {
                label: 'Memória %',
                data: (sysData.memory || []).map(p => p.value),
                borderColor: '#a78bfa',
                backgroundColor: 'rgba(167, 139, 250, 0.1)',
                fill: true,
            },
        ], {
            yFormat: v => v + '%',
        });

        // PPPoE chart (if available)
        if (sysData.pppoe && sysData.pppoe.length > 0) {
            document.getElementById('pppoe-card').style.display = 'block';
            document.getElementById('pppoe-chart-card').style.display = 'block';
            
            // Set current online users in card
            const latestPPPoE = sysData.pppoe[sysData.pppoe.length - 1].value;
            document.getElementById('pppoe-value').textContent = latestPPPoE;

            const pppoeLabels = sysData.pppoe.map(p => p.time);
            createLineChart('chart-pppoe', pppoeLabels, [
                {
                    label: 'PPPoE Online',
                    data: sysData.pppoe.map(p => p.value),
                    borderColor: '#f43f5e',
                    backgroundColor: 'rgba(244, 63, 94, 0.1)',
                    fill: true,
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
                borderColor: '#34d399',
                backgroundColor: 'rgba(52, 211, 153, 0.1)',
                fill: true,
            },
            {
                label: 'Packet Loss %',
                data: (pingData.packet_loss || []).map(p => p.value),
                borderColor: '#f87171',
                backgroundColor: 'rgba(248, 113, 113, 0.1)',
                fill: true,
                yAxisID: 'y1',
            },
        ]);
    }
}

// ============================================
// Interface Traffic Chart (Modal)
// ============================================
async function showTrafficChart(deviceId, ifIndex, ifDescr) {
    document.getElementById('traffic-modal-title').textContent = `Tráfego — ${ifDescr}`;
    document.getElementById('traffic-modal').classList.add('active');

    // Get active period
    const activeBtn = document.querySelector('#period-selector .chart-period.active');
    const period = activeBtn ? activeBtn.dataset.period : '1h';

    const data = await api(`/api/metrics/traffic?device_id=${deviceId}&if_index=${ifIndex}&period=${period}`);
    if (data) {
        const labels = (data.in_bps || []).map(p => p.time);
        createLineChart('chart-traffic-detail', labels, [
            {
                label: '↓ In',
                data: (data.in_bps || []).map(p => p.value),
                borderColor: '#34d399',
                backgroundColor: 'rgba(52, 211, 153, 0.1)',
                fill: true,
            },
            {
                label: '↑ Out',
                data: (data.out_bps || []).map(p => p.value),
                borderColor: '#38bdf8',
                backgroundColor: 'rgba(56, 189, 248, 0.1)',
                fill: true,
            },
        ], {
            yFormat: v => formatBps(v),
        });
    }
}
