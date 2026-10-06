.PHONY: tidy build run dev test lint setup docker-up docker-down

APP=feedback-api
PORTS?=8080,5173

# 清理占用开发端口的残留进程（避免旧服务未重启导致端口占用/代码不生效）
kill-ports:
	@for p in $$(echo $(PORTS) | tr ',' ' '); do \
		pids=$$(lsof -ti :$$p 2>/dev/null); \
		if [ -n "$$pids" ]; then \
			echo "kill :$$p -> $$pids"; \
			kill $$pids 2>/dev/null || true; \
		fi; \
	done

tidy:
	go mod tidy

build:
	go build -o bin/$(APP) ./cmd/api

run: kill-ports
	go run ./cmd/api

# 同时启动后端(:8080)和前端开发服务器(:5173)
dev: kill-ports
	@cd web && npm install --silent 2>/dev/null || true
	@make -j2 run web-dev

web-dev:
	cd web && npm run dev

test:
	go test -race -cover $(PKG)

lint:
	gofmt -l . | tee /tmp/gofmt.out && test ! -s /tmp/gofmt.out
	go vet $(PKG)

setup:
	go run ./cmd/setup

docker-up:
	docker compose up -d

docker-down:
	docker compose down
