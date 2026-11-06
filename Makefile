.PHONY: help up down db-init migrate migrate-auto migrate-down migrate-test schema-check backend frontend fmt lint clean

help:
	@echo "make up        起 postgres + redis"
	@echo "make down      停服务并保留数据"
	@echo "make db-init   开启 pgvector 扩展"
	@echo "make migrate      按 SQL 文件建库（backend/migrate/*.up.sql）"
	@echo "make migrate-auto 用 GORM AutoMigrate 建库（本地快速迭代）"
	@echo "make migrate-down 回滚最新一版"
	@echo "make schema-check SQL 静态校验（语法/建表顺序/列存在性）"
	@echo "make migrate-test 连真实库跑迁移集成测试（需先 make up）"
	@echo "make backend   本地起后端 (:8080)"
	@echo "make frontend  本地起前端 (:5173)"
	@echo "make fmt       格式化 go / frontend"
	@echo "make clean     停服务并清空数据卷（会丢数据）"

up:
	docker compose up -d

down:
	docker compose down

db-init:
	bash scripts/init-db.sh

migrate:
	cd backend && go run ./cmd/migrate --mode=sql

migrate-auto:
	cd backend && go run ./cmd/migrate --mode=auto

migrate-down:
	cd backend && go run ./cmd/migrate --mode=sql --down

schema-check:
	python3 scripts/check_schema.py backend/migrate/000001_init.up.sql

migrate-test:
	cd backend && NODAL_TEST_DSN="$${NODAL_TEST_DSN:-postgres://nodal:nodal_dev_pwd@localhost:5432/nodal?sslmode=disable}" \
		go test ./migrate -count=1 -run RealDB -v

backend:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm run dev

fmt:
	cd backend && go fmt ./...
	cd frontend && npm run format

lint:
	cd backend && go vet ./...
	cd frontend && npm run lint

clean:
	docker compose down -v
