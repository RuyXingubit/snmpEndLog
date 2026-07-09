-- Migration 010: AI Analysis Sessions
-- Stores AI chat sessions with context from log analysis.

CREATE TABLE IF NOT EXISTS ai_sessions (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER,
    title       VARCHAR(255) NOT NULL DEFAULT 'Nova Análise',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_messages (
    id          SERIAL PRIMARY KEY,
    session_id  INTEGER NOT NULL REFERENCES ai_sessions(id) ON DELETE CASCADE,
    role        VARCHAR(16) NOT NULL,  -- 'user', 'assistant', 'context'
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_messages_session ON ai_messages (session_id, created_at);
