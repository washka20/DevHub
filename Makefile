.PHONY: dev dev-go dev-vue build run clean

## Запуск в режиме разработки (Go + Vite параллельно)
dev:
	@echo "Starting DevHub in dev mode..."
	@make -j2 dev-go dev-vue

## Go сервер (dev mode)
dev-go:
	@cd cmd && go run . -dev

## Vite dev server
dev-vue:
	@cd frontend && npm run dev

## Сборка production бинарника
build:
	@echo "Building frontend..."
	@cd frontend && npm run build
	@echo "Building Go binary..."
	@cd cmd && go build -o ../devhub .
	@echo "Done! Run: ./devhub"

## Запуск production бинарника
run: build
	@./devhub

## Очистка
clean:
	@rm -f devhub
	@rm -rf frontend/dist
