#!/usr/bin/env python3
"""SQL 迁移静态校验（无需真实数据库）。

用法：python3 scripts/check_schema.py backend/migrate/000001_init.up.sql

校验项：
1. 语法：用 libpg_query（pglast）按 PostgreSQL 真实语法解析每一条语句
2. 建表顺序：外键引用的表必须在此前已创建（PG 不支持前向引用）
3. 列存在性：外键列、索引列必须真实存在于目标表
4. 覆盖度：PRD 要求的 9 张核心表必须全部出现
"""
import sys
from collections import defaultdict

import pglast
from pglast import ast
from pglast.enums import ConstrType

REQUIRED_TABLES = {
    "users", "canvases", "agent_templates", "canvas_nodes", "canvas_edges",
    "knowledge_files", "knowledge_chunks", "canvas_likes", "canvas_comments",
}


def table_of(node):
    try:
        return node.relation.relname
    except AttributeError:
        return None


def main(path: str) -> int:
    sql = open(path, encoding="utf-8").read()

    try:
        stmts = pglast.parse_sql(sql)
    except Exception as e:  # noqa: BLE001
        print(f"[FAIL] 语法解析失败: {e}")
        return 1
    print(f"[OK] 语法解析通过：{len(stmts)} 条语句")

    tables = {}          # name -> {columns: set()}
    created_order = []   # 建表顺序
    errors = []

    for raw in stmts:
        node = raw.stmt
        if isinstance(node, ast.CreateStmt):
            name = node.relation.relname
            cols = {}
            for elt in node.tableElts:
                if isinstance(elt, ast.ColumnDef):
                    cols[elt.colname] = elt
            tables[name] = cols
            created_order.append(name)

            # 表级外键：被引用表必须已存在
            for elt in node.tableElts:
                if isinstance(elt, ast.Constraint) and elt.contype == ConstrType.CONSTR_FOREIGN:
                    ref = elt.pktable.relname
                    if ref not in tables:
                        errors.append(f"{name}: 外键引用表 {ref} 尚未创建（PG 不支持前向引用）")
                    for k in elt.fk_attrs or []:
                        if k.sval not in cols:
                            errors.append(f"{name}: 外键列 {k.sval} 在表中不存在")
                    # 引用的列必须真实存在于被引用表
                    for k in elt.pk_attrs or []:
                        if ref in tables and k.sval not in tables[ref]:
                            errors.append(f"{name}: 外键引用了 {ref}.{k.sval}，该列不存在")

        elif isinstance(node, ast.IndexStmt):
            tbl = table_of(node)
            if tbl not in tables:
                errors.append(f"索引 {node.idxname}: 表 {tbl} 不存在")
                continue
            for p in node.indexParams:
                col = getattr(p, "name", None)
                if col and col not in tables[tbl]:
                    errors.append(f"索引 {node.idxname}: 列 {col} 在 {tbl} 中不存在")

        elif isinstance(node, ast.AlterTableStmt):
            tbl = table_of(node)
            if tbl not in tables:
                errors.append(f"ALTER TABLE: 表 {tbl} 不存在")

    if errors:
        print("[FAIL] 结构问题：")
        for e in errors:
            print("   -", e)
        return 1

    missing = REQUIRED_TABLES - set(tables)
    if missing:
        print(f"[FAIL] 缺少表: {sorted(missing)}")
        return 1

    print(f"[OK] 建表顺序合规：{' -> '.join(created_order)}")
    print(f"[OK] 覆盖 PRD 全部 {len(REQUIRED_TABLES)} 张核心表")

    # 附带报告：每张表的列数
    for t in created_order:
        print(f"     - {t}: {len(tables[t])} 列")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1] if len(sys.argv) > 1 else "backend/migrate/000001_init.up.sql"))
