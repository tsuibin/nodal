-- 由数据库超级用户执行一次：安装 pgvector 扩展。
-- 普通应用账号通常没有 CREATE EXTENSION 权限，因此与表结构脚本分开。
CREATE EXTENSION IF NOT EXISTS vector;
