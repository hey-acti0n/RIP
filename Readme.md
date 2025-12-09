# UltraRezina Backend API

Backend API для системы расчета резинотехнических изделий на Go.

## Структура проекта

- `main.go` - точка входа приложения
- `cmd/` - дополнительные команды
- `internal/` - внутренние пакеты приложения
  - `api/rest/` - REST API handlers
  - `app/` - бизнес-логика
    - `models/` - модели данных
    - `repository/` - работа с БД
    - `service/` - бизнес-логика
    - `middleware/` - middleware для аутентификации
    - `redis/` - клиент Redis
- `docs/` - Swagger документация
- `static/` - статические файлы
- `templates/` - HTML шаблоны

## Запуск

### С помощью скриптов:
- `./run_api.sh` - запуск только API
- `./run_web.sh` - запуск веб-сервера
- `./run_full.sh` - полный запуск

### С помощью Docker:
```bash
docker compose up -d
```

## API Документация

Swagger документация доступна по адресу: `http://localhost:8080/swagger/`

## Конфигурация

- Порт по умолчанию: `8080`
- База данных: PostgreSQL
- Redis: для кэширования и сессий (опционально)

