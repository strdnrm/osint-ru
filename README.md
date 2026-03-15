# OSINT-RU

**RU** | [EN](#english)

Веб-приложение для автоматического поиска по открытым государственным реестрам Российской Федерации. Вводишь известные данные о человеке — ФИО, дату рождения, ИНН — приложение параллельно опрашивает все подключённые источники и собирает результаты на одной странице.

> Используются **только публичные данные**. Никаких закрытых баз, утечек или платных API. Все запросы идут напрямую к официальным государственным сайтам (`.gov.ru`, `nalog.ru`, `fedresurs.ru`).

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)
![No deps](https://img.shields.io/badge/зависимости-0-brightgreen?style=flat-square)
![Sources](https://img.shields.io/badge/источников-6-blue?style=flat-square)
![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)

---

## Что ищет

| Источник | Автоматически | Правовая основа |
|---|:---:|---|
| **ФССП** — исполнительные производства, долги | ✅ | ФЗ №229-ФЗ, ст. 6.1 |
| **ФНС ЕГРЮЛ/ЕГРИП** — ИП и юрлица на физлицо | ✅ | ФЗ №129-ФЗ |
| **ФНС** — поиск ИНН по ФИО + дате рождения, валидация | ✅ | НК РФ, ст. 84 |
| **Федресурс** — процедуры банкротства физлиц | ✅ | ФЗ №127-ФЗ |
| **ГАС Правосудие** — судебные дела всех инстанций | 🔗 ссылки | ФЗ №262-ФЗ |
| **Росреестр, КАД, ФАС, Росфинмониторинг, ФНС-дисквал.** | 🔗 ссылки | Открытые данные |

Для источников с пометкой 🔗 приложение формирует прямые ссылки с предзаполненными параметрами — открываешь и сразу видишь результат.

---

## Запуск

### Требования

- Go 1.22 или новее — [скачать](https://go.dev/dl/)

### Из исходников

```bash
git clone https://github.com/YOUR_USERNAME/osint-ru.git
cd osint-ru
go run main.go
```

Открыть в браузере: **http://localhost:8080**

### Собрать бинарник

```bash
go build -ldflags="-s -w" -o osint-ru main.go
./osint-ru
```

Статика встроена в бинарник через `//go:embed` — один файл, никаких папок рядом не нужно.

### Изменить порт

```bash
PORT=9000 ./osint-ru
```

### Docker

```bash
docker build -t osint-ru .
docker run -p 8080:8080 osint-ru
```

<details>
<summary>Dockerfile</summary>

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags="-s -w" -o osint-ru main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/osint-ru .
EXPOSE 8080
CMD ["./osint-ru"]
```

</details>

---

## Структура проекта

```
osint-ru/
├── main.go                   # HTTP-сервер, embed статики
├── go.mod
├── internal/
│   ├── models/models.go      # типы: SearchQuery, SourceResult, Record, Field
│   ├── handlers/search.go    # POST /api/search, GET /api/health
│   └── sources/
│       ├── fssp.go           # ФССП
│       ├── fns.go            # ФНС ЕГРЮЛ/ЕГРИП
│       ├── inn.go            # ФНС ИНН
│       ├── fedresurs.go      # Федресурс
│       ├── sudrf.go          # ГАС Правосудие
│       ├── rosreestr.go      # Росреестр
│       └── gosuslugi.go      # агрегатор прямых ссылок
└── static/
    ├── index.html
    ├── css/style.css
    └── js/app.js
```

---

## Добавить новый источник

1. Создать `internal/sources/mysource.go`, реализовать метод:
```go
func (s *MySource) Search(q models.SearchQuery) models.SourceResult
```
2. Добавить в список в `internal/handlers/search.go`:
```go
searchers := []Searcher{
    // ...
    sources.NewMySource(),
}
```
Всё — источник автоматически включается в параллельный поиск.

---

## Правовая база

Приложение работает только с теми сведениями, публикацию которых **государство обязано обеспечивать** по федеральным законам:

- **ФЗ №229-ФЗ, ст. 6.1** — ФССП обязана вести и публиковать банк данных исполнительных производств в открытом доступе
- **ФЗ №129-ФЗ** — сведения из ЕГРЮЛ и ЕГРИП являются открытыми и общедоступными
- **ФЗ №262-ФЗ** — суды обязаны публиковать тексты судебных актов в сети Интернет
- **ФЗ №127-ФЗ** — сообщения о банкротстве подлежат обязательному раскрытию в публичном реестре

Приложение не нарушает ФЗ №152 «О персональных данных», так как оперирует исключительно теми данными, которые государство само раскрывает в публичных реестрах.

---

## License

MIT

---
---

# English

<a name="english"></a>

A web application for automated search across open Russian government registries. Enter what you know about a person — full name, date of birth, INN (tax ID) — and the app queries all connected sources in parallel, presenting results on a single page.

> **Only public data is used.** No private databases, no leaked data, no paid APIs. All requests go directly to official government websites (`.gov.ru`, `nalog.ru`, `fedresurs.ru`).

---

## What it searches

| Source | Auto | Legal basis |
|---|:---:|---|
| **FSSP** — enforcement proceedings, debts | ✅ | Federal Law №229-FZ, Art. 6.1 |
| **FTS EGRUL/EGRIP** — sole proprietorships and legal entities registered to a person | ✅ | Federal Law №129-FZ |
| **FTS** — find INN by full name + date of birth, INN validation | ✅ | Tax Code, Art. 84 |
| **Fedresurs** — personal bankruptcy proceedings | ✅ | Federal Law №127-FZ |
| **GAS Pravosudie** — court cases, all jurisdictions | 🔗 links | Federal Law №262-FZ |
| **Rosreestr, CAD, FAS, Rosfinmonitoring, FTS disqualified** | 🔗 links | Open data |

Sources marked 🔗 generate pre-filled direct links — open and see results immediately.

---

## Running

### Requirements

- Go 1.22 or later — [download](https://go.dev/dl/)

### From source

```bash
git clone https://github.com/YOUR_USERNAME/osint-ru.git
cd osint-ru
go run main.go
```

Open in browser: **http://localhost:8080**

### Build binary

```bash
go build -ldflags="-s -w" -o osint-ru main.go
./osint-ru
```

Static assets are embedded into the binary via `//go:embed` — single file, no folders needed alongside it.

### Change port

```bash
PORT=9000 ./osint-ru
```

### Docker

```bash
docker build -t osint-ru .
docker run -p 8080:8080 osint-ru
```

<details>
<summary>Dockerfile</summary>

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags="-s -w" -o osint-ru main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/osint-ru .
EXPOSE 8080
CMD ["./osint-ru"]
```

</details>

---

## Project structure

```
osint-ru/
├── main.go                   # HTTP server, static embed
├── go.mod
├── internal/
│   ├── models/models.go      # SearchQuery, SourceResult, Record, Field
│   ├── handlers/search.go    # POST /api/search, GET /api/health
│   └── sources/
│       ├── fssp.go           # FSSP enforcement proceedings
│       ├── fns.go            # FTS EGRUL/EGRIP
│       ├── inn.go            # FTS INN lookup & validation
│       ├── fedresurs.go      # Fedresurs bankruptcy registry
│       ├── sudrf.go          # GAS Pravosudie court portal
│       ├── rosreestr.go      # Rosreestr real estate
│       └── gosuslugi.go      # aggregated direct links
└── static/
    ├── index.html
    ├── css/style.css
    └── js/app.js
```

---

## Adding a new source

1. Create `internal/sources/mysource.go`, implement:
```go
func (s *MySource) Search(q models.SearchQuery) models.SourceResult
```
2. Register in `internal/handlers/search.go`:
```go
searchers := []Searcher{
    // ...
    sources.NewMySource(),
}
```
The source is automatically included in the parallel search.

---

## Legal basis

The application works exclusively with data that **the state is legally required to publish**:

- **Federal Law №229-FZ, Art. 6.1** — FSSP is required to maintain and publish the enforcement proceedings database in open access
- **Federal Law №129-FZ** — EGRUL/EGRIP records are open and publicly available
- **Federal Law №262-FZ** — courts are required to publish the texts of judicial acts on the Internet
- **Federal Law №127-FZ** — bankruptcy notices must be disclosed in a public registry

The application does not violate Federal Law №152-FZ "On Personal Data", as it operates exclusively with data that the state itself discloses in public registries.

---

## License

MIT
