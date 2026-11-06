-- Nodal 模块 1 · 初始迁移（down）
-- 逆序删除，先删引用方再删被引用方。
-- 注意：不 DROP EXTENSION vector / pgcrypto —— 扩展可能是同库其他业务在用。

DROP TABLE IF EXISTS canvas_comments;
DROP TABLE IF EXISTS canvas_likes;
DROP TABLE IF EXISTS knowledge_chunks;
DROP TABLE IF EXISTS knowledge_files;
DROP TABLE IF EXISTS canvas_edges;
DROP TABLE IF EXISTS canvas_nodes;
DROP TABLE IF EXISTS agent_templates;
DROP TABLE IF EXISTS canvases;
DROP TABLE IF EXISTS users;
