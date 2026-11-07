-- Nodal PostgreSQL 16 + pgvector 初始化脚本（参考）
-- 运行期仍会执行 GORM AutoMigrate；本脚本用于空库首次部署、审阅表结构或在托管环境预建表。
-- 前置：由超级用户先执行 0000_extensions.sql（CREATE EXTENSION vector）。
-- 本脚本应以应用数据库账号执行，保证表/序列归应用账号所有。
-- 注意：embedding 维度与 NODAL_EMBED_DIM 保持一致，默认 1536。

CREATE TABLE IF NOT EXISTS users (
    id            bigserial PRIMARY KEY,
    email         varchar(190) NOT NULL,
    password_hash varchar(255) NOT NULL,
    nickname      varchar(32)  NOT NULL DEFAULT '',
    avatar_url    varchar(512) NOT NULL DEFAULT '',
    status        smallint     NOT NULL DEFAULT 1,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    updated_at    timestamptz  NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users (status);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);

CREATE TABLE IF NOT EXISTS guest_sessions (
    id           varchar(36) PRIMARY KEY,
    client_ip    varchar(64)  NOT NULL DEFAULT '',
    user_agent   varchar(255) NOT NULL DEFAULT '',
    created_at   timestamptz  NOT NULL DEFAULT now(),
    expires_at   timestamptz  NOT NULL,
    last_seen_at timestamptz  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_guest_sessions_expires_at ON guest_sessions (expires_at);

CREATE TABLE IF NOT EXISTS canvases (
    id             bigserial PRIMARY KEY,
    owner_key      varchar(80)  NOT NULL,
    title          varchar(120) NOT NULL,
    description    varchar(500) NOT NULL DEFAULT '',
    nodes          jsonb        NOT NULL DEFAULT '[]'::jsonb,
    edges          jsonb        NOT NULL DEFAULT '[]'::jsonb,
    viewport       jsonb        NOT NULL DEFAULT '{"x":0,"y":0,"zoom":1}'::jsonb,
    tags           jsonb        NOT NULL DEFAULT '[]'::jsonb,
    is_public      boolean      NOT NULL DEFAULT false,
    share_mode     varchar(16)  NOT NULL DEFAULT 'private',
    forked_from    bigint,
    node_count     integer      NOT NULL DEFAULT 0,
    edge_count     integer      NOT NULL DEFAULT 0,
    published_at   timestamptz,
    like_count     integer      NOT NULL DEFAULT 0,
    comment_count  integer      NOT NULL DEFAULT 0,
    created_at     timestamptz  NOT NULL DEFAULT now(),
    updated_at     timestamptz  NOT NULL DEFAULT now(),
    deleted_at     timestamptz
);
CREATE INDEX IF NOT EXISTS idx_canvases_owner ON canvases (owner_key);
CREATE INDEX IF NOT EXISTS idx_canvases_public ON canvases (is_public);
CREATE INDEX IF NOT EXISTS idx_canvases_share_mode ON canvases (share_mode);
CREATE INDEX IF NOT EXISTS idx_canvases_forked_from ON canvases (forked_from);
CREATE INDEX IF NOT EXISTS idx_canvases_published_at ON canvases (published_at);
CREATE INDEX IF NOT EXISTS idx_canvases_deleted_at ON canvases (deleted_at);

CREATE TABLE IF NOT EXISTS agent_templates (
    id            bigserial PRIMARY KEY,
    owner_key     varchar(80)  NOT NULL,
    name          varchar(80)  NOT NULL,
    description   varchar(300) NOT NULL DEFAULT '',
    system_prompt text         NOT NULL,
    model_id      varchar(80)  NOT NULL,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    updated_at    timestamptz  NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_agents_owner_name ON agent_templates (owner_key, name);

CREATE TABLE IF NOT EXISTS model_credentials (
    id          bigserial PRIMARY KEY,
    owner_key   varchar(80)  NOT NULL,
    provider    varchar(32)  NOT NULL,
    api_key_enc text         NOT NULL,
    key_hint    varchar(24)  NOT NULL DEFAULT '',
    base_url    varchar(255) NOT NULL DEFAULT '',
    created_at  timestamptz  NOT NULL DEFAULT now(),
    updated_at  timestamptz  NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_credentials_owner_provider ON model_credentials (owner_key, provider);

CREATE TABLE IF NOT EXISTS knowledge_files (
    id          bigserial PRIMARY KEY,
    owner_key   varchar(80)  NOT NULL,
    name        varchar(160) NOT NULL,
    ext         varchar(16)  NOT NULL,
    mime        varchar(96)  NOT NULL DEFAULT '',
    size        bigint       NOT NULL DEFAULT 0,
    stored_name varchar(64)  NOT NULL,
    status      varchar(16)  NOT NULL DEFAULT 'processing',
    chunk_count integer      NOT NULL DEFAULT 0,
    error_msg   varchar(300) NOT NULL DEFAULT '',
    created_at  timestamptz  NOT NULL DEFAULT now(),
    updated_at  timestamptz  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_knowledge_files_owner ON knowledge_files (owner_key);

CREATE TABLE IF NOT EXISTS knowledge_chunks (
    id             bigserial PRIMARY KEY,
    owner_key      varchar(80) NOT NULL,
    file_id        bigint      NOT NULL,
    chunk_index    integer     NOT NULL DEFAULT 0,
    content        text        NOT NULL,
    embedding_json text        NOT NULL DEFAULT '',
    embedding      vector(1536),
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_owner ON knowledge_chunks (owner_key);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_file ON knowledge_chunks (file_id);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_embedding
    ON knowledge_chunks USING hnsw (embedding vector_cosine_ops);

CREATE TABLE IF NOT EXISTS canvas_likes (
    id         bigserial PRIMARY KEY,
    canvas_id  bigint      NOT NULL,
    user_key   varchar(80) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS uk_canvas_likes_canvas_user ON canvas_likes (canvas_id, user_key);

CREATE TABLE IF NOT EXISTS canvas_comments (
    id         bigserial PRIMARY KEY,
    canvas_id  bigint      NOT NULL,
    user_key   varchar(80) NOT NULL,
    user_id    bigint      NOT NULL,
    content    varchar(1000) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_canvas_comments_canvas ON canvas_comments (canvas_id);
CREATE INDEX IF NOT EXISTS idx_canvas_comments_user ON canvas_comments (user_key);
CREATE INDEX IF NOT EXISTS idx_canvas_comments_uid ON canvas_comments (user_id);
CREATE INDEX IF NOT EXISTS idx_canvas_comments_deleted_at ON canvas_comments (deleted_at);

CREATE TABLE IF NOT EXISTS canvas_share_links (
    id          bigserial PRIMARY KEY,
    canvas_id   bigint      NOT NULL,
    token       varchar(64) NOT NULL,
    permission  varchar(16) NOT NULL DEFAULT 'read',
    created_by  varchar(80) NOT NULL,
    enabled     boolean     NOT NULL DEFAULT true,
    expires_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_canvas_share_links_canvas ON canvas_share_links (canvas_id);
CREATE UNIQUE INDEX IF NOT EXISTS uk_canvas_share_links_token ON canvas_share_links (token);
