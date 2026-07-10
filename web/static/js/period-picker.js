/**
 * nms — Period Picker Component
 * Adds a "Personalizado" option to period <select> elements
 * and shows inline datetime-local inputs when selected.
 *
 * Usage:
 *   initPeriodPicker('log-period', { onChange: searchLogs })
 *   const params = getPeriodParams('log-period')
 *   // Returns: { period: '1h' } or { period: 'custom', start: '...', end: '...' }
 */

'use strict';

/**
 * Initialize a period picker on a <select> element.
 * @param {string} selectId - The id of the <select> element.
 * @param {Object} options - Options.
 * @param {Function} [options.onChange] - Callback when period changes.
 */
function initPeriodPicker(selectId, options = {}) {
    const select = document.getElementById(selectId);
    if (!select) return;

    // Add "Personalizado" option if not already present
    if (!select.querySelector('option[value="custom"]')) {
        const opt = document.createElement('option');
        opt.value = 'custom';
        opt.textContent = '📅 Personalizado';
        select.appendChild(opt);
    }

    // Create date range container
    const container = document.createElement('div');
    container.className = 'custom-date-range';
    container.id = selectId + '-custom-range';
    container.style.display = 'none';
    container.innerHTML = `
        <label class="custom-date-label">De:
            <input type="datetime-local" class="form-input custom-date-input" id="${selectId}-start">
        </label>
        <label class="custom-date-label">Até:
            <input type="datetime-local" class="form-input custom-date-input" id="${selectId}-end">
        </label>
    `;

    // Insert after the select's parent filter bar
    select.parentElement.insertAdjacentElement('afterend', container);

    // Set default values: start = 24h ago, end = now
    const now = new Date();
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);
    document.getElementById(selectId + '-start').value = toDatetimeLocal(yesterday);
    document.getElementById(selectId + '-end').value = toDatetimeLocal(now);

    // Toggle visibility on select change
    select.addEventListener('change', () => {
        const isCustom = select.value === 'custom';
        container.style.display = isCustom ? 'flex' : 'none';
        if (options.onChange) options.onChange();
    });

    // Trigger search on date change
    const startInput = document.getElementById(selectId + '-start');
    const endInput = document.getElementById(selectId + '-end');
    if (options.onChange) {
        startInput.addEventListener('change', options.onChange);
        endInput.addEventListener('change', options.onChange);
    }
}

/**
 * Get period parameters for API calls.
 * @param {string} selectId - The id of the <select> element.
 * @returns {Object} - { period: '1h' } or { period: 'custom', start: '...', end: '...' }
 */
function getPeriodParams(selectId) {
    const select = document.getElementById(selectId);
    if (!select) return { period: '1h' };

    if (select.value === 'custom') {
        const start = document.getElementById(selectId + '-start').value;
        const end = document.getElementById(selectId + '-end').value;
        return { period: 'custom', start, end };
    }

    return { period: select.value };
}

/**
 * Convert a Date to datetime-local input value format.
 * @param {Date} date
 * @returns {string} - "YYYY-MM-DDTHH:MM"
 */
function toDatetimeLocal(date) {
    const y = date.getFullYear();
    const m = String(date.getMonth() + 1).padStart(2, '0');
    const d = String(date.getDate()).padStart(2, '0');
    const h = String(date.getHours()).padStart(2, '0');
    const min = String(date.getMinutes()).padStart(2, '0');
    return `${y}-${m}-${d}T${h}:${min}`;
}
