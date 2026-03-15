.PHONY: build run dev clean lint test build-all build-linux build-windows build-mac

## Собрать бинарник под текущую платформу
build:
	go build -ldflags="-s -w" -o osint-ru main.go

## Запустить без сборки (режим разработки)
run: dev
dev:
	go run main.go

## Проверить код
lint:
	go vet ./...

## Запустить тесты
test:
	go test ./...

## Удалить бинарники
clean:
	rm -f osint-ru osint-ru-linux-amd64 osint-ru.exe osint-ru-mac-arm64

## Кросс-компиляция под Linux (x86_64)
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o osint-ru-linux-amd64 main.go

## Кросс-компиляция под Windows
build-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o osint-ru.exe main.go

## Кросс-компиляция под macOS Apple Silicon
build-mac:
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o osint-ru-mac-arm64 main.go

## Собрать под все платформы
build-all: build-linux build-windows build-mac
	@echo "Готово:"
	@ls -lh osint-ru-linux-amd64 osint-ru.exe osint-ru-mac-arm64 2>/dev/null || true
