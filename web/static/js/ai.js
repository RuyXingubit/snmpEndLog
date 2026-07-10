/**
 * nms — AI Analysis JavaScript
 * Manages AI sessions, context, and chat interactions.
 */

'use strict';

let currentSessionId = null;
let contextLogCount = 0;

// ============================================
// Session Management
// ============================================
async function loadSessions() {
    const sessions = await api('/api/ai/sessions');
    if (!sessions) return;

    const list = document.getElementById('session-list');

    if (!sessions || sessions.length === 0) {
        list.innerHTML = `
            <div class="empty-state" style="padding: 1rem;">
                <p class="text-muted" style="font-size: 0.8rem;">Nenhuma sessão. Crie uma nova para começar.</p>
            </div>
        `;
        return;
    }

    let html = '';
    sessions.forEach(s => {
        const active = s.id === currentSessionId ? ' active' : '';
        const time = formatTimeShort(s.updated_at);
        html += `
            <div class="ai-session-item${active}" onclick="selectSession(${s.id})" data-id="${s.id}">
                <div class="ai-session-title">${escapeHtml(s.title)}</div>
                <div class="ai-session-time">${time}</div>
                <button class="ai-session-delete" onclick="event.stopPropagation(); deleteSession(${s.id})" title="Excluir">✕</button>
            </div>
        `;
    });

    list.innerHTML = html;
}

async function createSession() {
    const result = await api('/api/ai/sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title: 'Análise ' + new Date().toLocaleDateString('pt-BR') }),
    });
    if (!result) return;

    currentSessionId = result.id;
    contextLogCount = 0;
    await loadSessions();
    selectSession(result.id);
}

async function deleteSession(id) {
    if (!confirm('Excluir esta sessão?')) return;

    await api(`/api/ai/sessions/${id}`, { method: 'DELETE' });

    if (currentSessionId === id) {
        currentSessionId = null;
        contextLogCount = 0;
        document.getElementById('ai-no-session').style.display = '';
        document.getElementById('ai-session-view').style.display = 'none';
    }

    await loadSessions();
}

async function selectSession(id) {
    currentSessionId = id;
    contextLogCount = 0;

    // Show session view
    document.getElementById('ai-no-session').style.display = 'none';
    const view = document.getElementById('ai-session-view');
    view.style.display = 'flex';

    // Update active state in sidebar
    document.querySelectorAll('.ai-session-item').forEach(el => {
        el.classList.toggle('active', parseInt(el.dataset.id) === id);
    });

    // Load messages
    await loadMessages();
}

// ============================================
// Messages
// ============================================
async function loadMessages() {
    if (!currentSessionId) return;

    const messages = await api(`/api/ai/sessions/${currentSessionId}/messages`);
    const container = document.getElementById('ai-messages');

    if (!messages || messages.length === 0) {
        container.innerHTML = `
            <div class="ai-welcome">
                <p>👋 Use os filtros acima para <strong>adicionar logs ao contexto</strong>, depois faça perguntas.</p>
            </div>
        `;
        updateContextInfo();
        return;
    }

    let html = '';
    messages.forEach(m => {
        if (m.role === 'context') {
            // Count logs in context
            const lines = m.content.split('\n').filter(l => l.startsWith('[')).length;
            contextLogCount += lines;
            html += `
                <div class="ai-msg ai-msg-context">
                    <div class="ai-msg-label">📎 Contexto adicionado</div>
                    <div class="ai-msg-detail">${lines} logs carregados</div>
                </div>
            `;
        } else if (m.role === 'user') {
            html += `
                <div class="ai-msg ai-msg-user">
                    <div class="ai-msg-content">${escapeHtml(m.content).replace(/\n/g, '<br>')}</div>
                </div>
            `;
        } else if (m.role === 'assistant') {
            html += `
                <div class="ai-msg ai-msg-assistant">
                    <div class="ai-msg-content ai-markdown">${renderMarkdown(m.content)}</div>
                </div>
            `;
        }
    });

    container.innerHTML = html;
    container.scrollTop = container.scrollHeight;
    updateContextInfo();
}

function updateContextInfo() {
    const info = document.getElementById('context-info');
    if (contextLogCount > 0) {
        info.textContent = `${contextLogCount} logs no contexto`;
    } else {
        info.textContent = '';
    }
}

// ============================================
// Clear Context
// ============================================
async function clearContext() {
    if (!currentSessionId) return;
    if (!confirm('Limpar todo o contexto e conversa desta sessão?')) return;

    await api(`/api/ai/sessions/${currentSessionId}/context`, { method: 'DELETE' });
    contextLogCount = 0;
    await loadMessages();
}

// ============================================
// Add Context
// ============================================
async function addContext() {
    if (!currentSessionId) {
        alert('Selecione ou crie uma sessão primeiro.');
        return;
    }

    const btn = document.getElementById('btn-add-context');
    btn.disabled = true;
    btn.textContent = '⏳ Carregando...';

    const body = {
        host: document.getElementById('ai-host').value,
        severity: document.getElementById('ai-severity').value,
        period: document.getElementById('ai-period').value,
    };

    const result = await api(`/api/ai/sessions/${currentSessionId}/context`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
    });

    btn.disabled = false;
    btn.textContent = '📎 Adicionar ao Contexto';

    if (result) {
        if (result.count > 0) {
            contextLogCount += result.count;
        }
        await loadMessages();
    }
}

// ============================================
// Ask AI
// ============================================
async function askAI() {
    if (!currentSessionId) return;

    const input = document.getElementById('ai-question');
    const question = input.value.trim();
    if (!question) return;

    const btn = document.getElementById('btn-ask');
    btn.disabled = true;
    btn.textContent = '⏳ Analisando...';
    input.disabled = true;

    // Immediately show user message (preserve line breaks)
    const container = document.getElementById('ai-messages');
    const escapedQuestion = escapeHtml(question).replace(/\n/g, '<br>');
    container.innerHTML += `
        <div class="ai-msg ai-msg-user">
            <div class="ai-msg-content">${escapedQuestion}</div>
        </div>
        <div class="ai-msg ai-msg-assistant" id="ai-loading">
            <div class="ai-msg-content"><div class="loading-spinner" style="margin: 0 auto;"></div></div>
        </div>
    `;
    container.scrollTop = container.scrollHeight;
    input.value = '';
    input.style.height = 'auto'; // Reset textarea height

    const result = await api(`/api/ai/sessions/${currentSessionId}/ask`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ question }),
    });

    // Remove loading indicator
    const loading = document.getElementById('ai-loading');
    if (loading) loading.remove();

    if (result && result.response) {
        container.innerHTML += `
            <div class="ai-msg ai-msg-assistant">
                <div class="ai-msg-content ai-markdown">${renderMarkdown(result.response)}</div>
            </div>
        `;
        container.scrollTop = container.scrollHeight;
    }

    btn.disabled = false;
    btn.textContent = 'Analisar';
    input.disabled = false;
    input.focus();
}

// ============================================
// Load Hosts for AI Filter
// ============================================
async function loadAIHosts() {
    const hosts = await api('/api/logs/hosts');
    if (!hosts) return;

    const select = document.getElementById('ai-host');
    hosts.forEach(host => {
        const opt = document.createElement('option');
        opt.value = host;
        opt.textContent = host;
        select.appendChild(opt);
    });
}

// ============================================
// Simple Markdown Renderer
// ============================================
function renderMarkdown(text) {
    if (!text) return '';

    // Escape HTML first
    let html = text
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');

    // Headers
    html = html.replace(/^### (.+)$/gm, '<h4>$1</h4>');
    html = html.replace(/^## (.+)$/gm, '<h3>$1</h3>');
    html = html.replace(/^# (.+)$/gm, '<h2>$1</h2>');

    // Bold and italic
    html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
    html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');

    // Inline code
    html = html.replace(/`([^`]+)`/g, '<code>$1</code>');

    // Code blocks
    html = html.replace(/```(\w*)\n([\s\S]*?)```/g, '<pre><code>$2</code></pre>');

    // Unordered lists
    html = html.replace(/^[-*] (.+)$/gm, '<li>$1</li>');
    html = html.replace(/(<li>.*<\/li>\n?)+/gs, '<ul>$&</ul>');

    // Ordered lists
    html = html.replace(/^\d+\. (.+)$/gm, '<li>$1</li>');

    // Line breaks (double newline = paragraph)
    html = html.replace(/\n\n/g, '</p><p>');
    html = '<p>' + html + '</p>';

    // Clean up empty paragraphs
    html = html.replace(/<p>\s*<\/p>/g, '');
    html = html.replace(/<p>\s*(<h[234]>)/g, '$1');
    html = html.replace(/(<\/h[234]>)\s*<\/p>/g, '$1');
    html = html.replace(/<p>\s*(<ul>)/g, '$1');
    html = html.replace(/(<\/ul>)\s*<\/p>/g, '$1');
    html = html.replace(/<p>\s*(<pre>)/g, '$1');
    html = html.replace(/(<\/pre>)\s*<\/p>/g, '$1');

    return html;
}

// ============================================
// Utility
// ============================================
function escapeHtml(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
}

// Textarea: Enter sends, Shift+Enter adds new line, auto-resize
document.addEventListener('DOMContentLoaded', () => {
    const input = document.getElementById('ai-question');
    if (!input) return;

    // Max height ~10 lines (10 * ~1.4em line-height * 14px base ≈ 200px)
    const MAX_HEIGHT = 200;

    function autoResize() {
        input.style.height = 'auto';
        input.style.height = Math.min(input.scrollHeight, MAX_HEIGHT) + 'px';
        // Show scrollbar only when content exceeds max
        input.style.overflowY = input.scrollHeight > MAX_HEIGHT ? 'auto' : 'hidden';
    }

    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            askAI();
        }
    });

    input.addEventListener('input', autoResize);
});
