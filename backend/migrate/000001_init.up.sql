-- Nodal 模块 1 · 初始迁移（up）
-- 对应 docs/PRD.md §7 数据模型。
-- 约定：
--   * 主键统一 uuid，默认 gen_random_uuid()（需 pgcrypto）
--   * 时间统一 timestamptz，写入用 UTC
--   * 画布快照存 canvases.content_json，canvas_nodes / canvas_edges 为关系化投影
--   * knowledge_chunks.embedding 为 pgvector 的 vector(1536)

-- ---------------------------------------------------------------------------
-- 0. 扩展
-- ---------------------------------------------------------------------------
CREATE EXTENSION IF NOT EXISTS pgcrypto;  -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS vector;    -- pgvector

-- ---------------------------------------------------------------------------
-- 1. 用户
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id            uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    email         varchar(255) NOT NULL,
    password_hash varchar(255) NOT NULL,
    nickname      varchar(64)  NOT NULL DEFAULT '',
    avatar_url    text,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    updated_at    timestamptz  NOT NULL DEFAULT now()
);

-- 邮箱唯一：写入前由应用层统一转小写，保证与 GORM 的 uniqueIndex 口径一致
CREATE UNIQUE INDEX IF NOT EXISTS uk_users_email ON users (email);

-- ---------------------------------------------------------------------------
-- 2. 画布
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS canvases (
    id               uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id         uuid         NOT NULL,
    title            varchar(200) NOT NULL DEFAULT '未命名画布',
    content_json     jsonb        NOT NULL DEFAULT '{"nodes":[],"edges":[],"viewport":{"x":0,"y":0,"zoom":1}}'::jsonb,
    share_mode       varchar(16)  NOT NULL DEFAULT 'private',
    is_public        boolean      NOT NULL DEFAULT false,
    forked_from      uuid,
    forked_from_node varchar(64),
    created_at       timestamptz  NOT NULL DEFAULT now(),
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT ck_canvases_share_mode CHECK (share_mode IN ('private', 'read', 'fork')),
    CONSTRAINT fk_canvases_owner FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_canvases_forked_from FOREIGN KEY (forked_from) REFERENCES canvases (id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_canvases_owner   ON canvases (owner_id);
CREATE INDEX IF NOT EXISTS idx_canvases_share   ON canvases (share_mode);
-- 广场列表：公开画布按更新时间倒序分页
CREATE INDEX IF NOT EXISTS idx_canvases_public  ON canvases (is_public, updated_at DESC) WHERE is_public = true;

-- ---------------------------------------------------------------------------
-- 3. Agent 模板
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS agent_templates (
    id            uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id      uuid,                                  -- null = 系统内置
    name          varchar(64)  NOT NULL,
    role          varchar(32)  NOT NULL,
    description   varchar(200) NOT NULL DEFAULT '',
    system_prompt text         NOT NULL DEFAULT '',
    model_id      varchar(64)  NOT NULL DEFAULT '',
    is_builtin    boolean      NOT NULL DEFAULT false,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    updated_at    timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT ck_agents_role CHECK (role IN ('explorer','verifier','summarizer','builder')),
    CONSTRAINT fk_agents_owner FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_agents_owner   ON agent_templates (owner_id);
CREATE INDEX IF NOT EXISTS idx_agents_role    ON agent_templates (role);
CREATE INDEX IF NOT EXISTS idx_agents_builtin ON agent_templates (is_builtin);
-- 内置模板按名字唯一，保证种子数据可重复执行
CREATE UNIQUE INDEX IF NOT EXISTS uk_agents_builtin_name ON agent_templates (name) WHERE is_builtin = true;

-- ---------------------------------------------------------------------------
-- 4. 节点（(canvas_id, node_id) 复合主键）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS canvas_nodes (
    canvas_id   uuid             NOT NULL,
    node_id     varchar(64)      NOT NULL,
    node_type   varchar(16)      NOT NULL DEFAULT 'text',
    position_x  double precision NOT NULL DEFAULT 0,
    position_y  double precision NOT NULL DEFAULT 0,
    data_json   jsonb            NOT NULL DEFAULT '{}'::jsonb,
    source      varchar(8)       NOT NULL DEFAULT 'user',
    agent_id    uuid,
    status      varchar(16)      NOT NULL DEFAULT 'idle',
    created_at  timestamptz      NOT NULL DEFAULT now(),
    updated_at  timestamptz      NOT NULL DEFAULT now(),
    PRIMARY KEY (canvas_id, node_id),
    CONSTRAINT ck_nodes_type   CHECK (node_type IN ('text','code','formula','file','image')),
    CONSTRAINT ck_nodes_source CHECK (source IN ('user','agent')),
    CONSTRAINT ck_nodes_status CHECK (status IN ('idle','streaming','changed','error')),
    CONSTRAINT fk_nodes_canvas FOREIGN KEY (canvas_id) REFERENCES canvases (id) ON DELETE CASCADE,
    CONSTRAINT fk_nodes_agent  FOREIGN KEY (agent_id) REFERENCES agent_templates (id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_nodes_type   ON canvas_nodes (node_type);
CREATE INDEX IF NOT EXISTS idx_nodes_source ON canvas_nodes (source);

-- ---------------------------------------------------------------------------
-- 5. 连线
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS canvas_edges (
    canvas_id      uuid        NOT NULL,
    edge_id        varchar(64) NOT NULL,
    source_node_id varchar(64) NOT NULL,
    target_node_id varchar(64) NOT NULL,
    relation       varchar(16) NOT NULL DEFAULT 'derives',
    label          varchar(32) NOT NULL DEFAULT '',
    data_json      jsonb       NOT NULL DEFAULT '{}'::jsonb,
    created_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (canvas_id, edge_id),
    CONSTRAINT ck_edges_relation CHECK (relation IN ('derives','refutes','supports','parallel','optimizes')),
    CONSTRAINT fk_edges_canvas   FOREIGN KEY (canvas_id) REFERENCES canvases (id) ON DELETE CASCADE,
    -- 复合外键：连线两端必须存在于同一画布，画布删节点时连线一并消失
    CONSTRAINT fk_edges_source FOREIGN KEY (canvas_id, source_node_id)
        REFERENCES canvas_nodes (canvas_id, node_id) ON DELETE CASCADE,
    CONSTRAINT fk_edges_target FOREIGN KEY (canvas_id, target_node_id)
        REFERENCES canvas_nodes (canvas_id, node_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_edges_source   ON canvas_edges (canvas_id, source_node_id);
CREATE INDEX IF NOT EXISTS idx_edges_target   ON canvas_edges (canvas_id, target_node_id);
CREATE INDEX IF NOT EXISTS idx_edges_relation ON canvas_edges (relation);

-- ---------------------------------------------------------------------------
-- 6. 知识库文件与切片
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS knowledge_files (
    id           uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id     uuid         NOT NULL,
    filename     varchar(255) NOT NULL,
    mime         varchar(64)  NOT NULL DEFAULT '',
    size_bytes   bigint       NOT NULL DEFAULT 0,
    storage_path text         NOT NULL DEFAULT '',
    status       varchar(16)  NOT NULL DEFAULT 'pending',
    chunk_count  integer      NOT NULL DEFAULT 0,
    created_at   timestamptz  NOT NULL DEFAULT now(),
    updated_at   timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT ck_kfiles_status CHECK (status IN ('pending','indexing','ready','failed')),
    CONSTRAINT ck_kfiles_size   CHECK (size_bytes >= 0),
    CONSTRAINT fk_kfiles_owner  FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_kfiles_owner  ON knowledge_files (owner_id);
CREATE INDEX IF NOT EXISTS idx_kfiles_status ON knowledge_files (status);

CREATE TABLE IF NOT EXISTS knowledge_chunks (
    id          uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id     uuid         NOT NULL,
    chunk_index integer      NOT NULL,
    content     text         NOT NULL,
    token_count integer      NOT NULL DEFAULT 0,
    embedding   vector(1536),
    created_at  timestamptz  NOT NULL DEFAULT now(),
    UNIQUE (file_id, chunk_index),
    CONSTRAINT ck_chunks_index CHECK (chunk_index >= 0),
    CONSTRAINT fk_chunks_file  FOREIGN KEY (file_id) REFERENCES knowledge_files (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_chunks_file ON knowledge_chunks (file_id);
-- HNSW + 余弦距离：RAG 召回用 vector_cosine_ops
CREATE INDEX IF NOT EXISTS idx_chunks_embedding
    ON knowledge_chunks USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);

-- ---------------------------------------------------------------------------
-- 7. 点赞与评论
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS canvas_likes (
    canvas_id  uuid        NOT NULL,
    user_id    uuid        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (canvas_id, user_id),
    CONSTRAINT fk_likes_canvas FOREIGN KEY (canvas_id) REFERENCES canvases (id) ON DELETE CASCADE,
    CONSTRAINT fk_likes_user   FOREIGN KEY (user_id)   REFERENCES users (id)    ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_likes_user ON canvas_likes (user_id);

CREATE TABLE IF NOT EXISTS canvas_comments (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    canvas_id  uuid        NOT NULL,
    user_id    uuid        NOT NULL,
    node_id    varchar(64),
    content    text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_comments_content CHECK (length(content) > 0),
    CONSTRAINT fk_comments_canvas FOREIGN KEY (canvas_id) REFERENCES canvases (id) ON DELETE CASCADE,
    CONSTRAINT fk_comments_user   FOREIGN KEY (user_id)   REFERENCES users (id)    ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_comments_canvas ON canvas_comments (canvas_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_comments_user   ON canvas_comments (user_id);

-- ---------------------------------------------------------------------------
-- 8. 内置 Agent 种子数据（PRD §4 F-08）
-- ---------------------------------------------------------------------------
INSERT INTO agent_templates (name, role, description, system_prompt, model_id, is_builtin)
VALUES
 ('探索者', 'explorer', '从当前节点发散更多可能性与分支',
  '你是 Nodal 的探索者。基于给定节点，提出 2-4 条值得继续推演的分支假设。每条假设要指出它的关键前提，不要下结论，不要替用户决策。',
  '', true),
 ('校验者', 'verifier', '挑出逻辑漏洞、反例与不成立的隐含前提',
  '你是 Nodal 的校验者。针对给定节点寻找逻辑漏洞、反例与隐含前提。每条质疑必须指出具体是哪一步推理不成立，给出可验证的检验方式。',
  '', true),
 ('总结者', 'summarizer', '把多条分支收拢成结构化结论',
  '你是 Nodal 的总结者。把画布上已有多条分支收拢为一份结构化结论：共识、分歧、待验证项三部分，每部分不超过三条。',
  '', true),
 ('构建者', 'builder', '把结论落成可直接使用的产物',
  '你是 Nodal 的构建者。把结论落成可直接使用的产物：代码、提纲、清单或方案骨架。产物要能直接执行或直接落地，不要停在描述层。',
  '', true)
ON CONFLICT DO NOTHING;
