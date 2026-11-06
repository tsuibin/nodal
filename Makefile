.PHONY: help up down db-init backend frontend fmt lint clean

help:
	@echo "make up        起 postgres + redis"
	@echo "make down      停服务并保留数据"
	@echo "make db-init   开启 pgvector 扩展"
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
