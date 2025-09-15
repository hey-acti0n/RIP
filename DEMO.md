# Демо-гайд: UltraRezina x PostgreSQL (ORM + RAW SQL)

## Подготовка
```bash
# 1) Запуск БД и сервисов
cd /Users/acti0n/Documents/RIP
docker compose up -d db

# 2) Запуск приложения
DATABASE_DSN="host=localhost user=postgres password=root dbname=rip port=5432 sslmode=disable TimeZone=UTC" \
  go run .
# Сервер слушает :8080
```

## Данные и роли (Adminer/Django)
- Таблицы: `services`, `requests`, `request_services`.
- В `services` должны быть записи (name, is_active=true). Поля темы: `density`, `thickness`, `material`.
- Ограничение статуса в `requests`: pending | in_progress | completed | rejected.

---

## Показ 3 страниц приложения
1) Каталог с поиском
- Открой `http://localhost:8080/?requestId=1` (requestId может быть пустым – создастся при добавлении).
- Введите строку в поле поиска и отправьте.
- Проверка: в Network нетворке браузера виден GET-рендер страницы и SSR-список карточек.

2) Детальная страница услуги
- Клик «Подробнее» на карточке.
- URL вида: `/detail/{id}?requestId=...`
- Видно характеристики из БД: `density`, `thickness`, `material`.

3) Страница корзины/расчёта
- Клик по иконке корзины или `Перейти в расчёт` → `/calc?requestId=...`
- Отображаются элементы корзины и расчёт.

---

## Операции с заявкой (ORM + RAW SQL)

### 1. Выполнить поиск (ORM)
- Контроллер: `MountORMRoutes → /orm/services`.
```text
GET /orm/services?q=mat&thickness=10
```
- ORM-запрос: фильтрация по `name` и `thickness` в таблице `services`.

### 2. Добавить две услуги в текущую заявку (ORM)
- На странице каталога нажать «Купить» два раза на разных товарах.
- Бизнес-логика:
  - При первом добавлении создаётся `requests` со статусом `pending` (если ещё нет).
  - Вставка в `request_services` с upsert по составному ключу `(request_id, service_id)`.
- Контроллер (SSR): `main.go → handleAdd` выполняет SQL:
```sql
INSERT INTO request_services (request_id, service_id, quantity)
VALUES ($1, $2, 1)
ON CONFLICT (request_id, service_id)
DO UPDATE SET quantity = request_services.quantity + 1;
```
- Карточка корзины в шапке увеличит число, на `/calc?requestId=...` отобразятся позиции.

### 3. Содержимое заявки (ORM)
- Открыть:
```text
GET /orm/request/current?userId=1
```
- Ответ JSON: `request` + `items` (с preload услуги).

### 4. Логическое удаление заявки (RAW SQL)
- Кнопка «Очистить корзину» на `/calc` отправляет POST `/clear`.
- Выполняется RAW SQL (без ORM):
```sql
UPDATE requests SET status = 'rejected' WHERE id = $1;
DELETE FROM request_services WHERE request_id = $1;
```
- Проверка запроса в БД (через psql/Adminer):
```sql
SELECT id, status, created_at FROM requests ORDER BY id DESC LIMIT 5;
SELECT * FROM request_services WHERE request_id = <id>;
```
- Поведение: страница расчёта очищена, статус → `rejected`.

### 5. Переход по URL удалённой заявки
```text
GET /orm/request?id=<deleted_id>
```
- Результат: 404 (удалённые `rejected` недоступны к просмотру).

### 6. Изменение полей в БД и показ в UI
- В Adminer измените тему полей:
```sql
UPDATE services SET density = 150.5, thickness = 12.0, material = 'EPDM'
WHERE id = <service_id>;
```
- Обновите страницу каталога/детальной — характеристики подтянутся в карточки и `detail`.

---

## Что показать в коде

### Модели (ORM)
- Файл: `db.go`
  - `DBService` (таблица `services`) — поля `density`, `thickness`, `material`.
  - `Request` (таблица `requests`).
  - `RequestService` (таблица `request_services`).
- Составной уникальный ключ M-M: в `RequestService` `primaryKey` на `(request_id, service_id)`.

### Контроллеры через ORM (4 шт.)
- Файл: `db.go → MountORMRoutes`
  1) `GET /orm/services` — поиск услуг.
  2) `POST /orm/request/add` — добавление услуги в черновик (создание pending при отсутствии).
  3) `GET /orm/request/current` — получение текущей заявки.
  4) `GET /orm/request` — получить заявку по id (без `rejected`).

### Удаление заявки через SQL UPDATE (без ORM)
- Файлы:
  - `main.go → handleClear` — `UPDATE requests SET status='rejected'` + `DELETE request_services`.
  - Отдельный ORM-роут `POST /orm/request/delete` (альтернатива) — также RAW SQL `UPDATE`.

---

## Проверка статусов заявки
- После первого добавления: `pending` (форсируется в `handleAdd`).
- После оформления на `/calc` (POST): `completed`.
- После очистки корзины: `rejected`.

Показ в БД:
```sql
SELECT id, status, created_at FROM requests ORDER BY id DESC LIMIT 10;
```

---

## Быстрые ссылки
- Каталог: `http://localhost:8080/?requestId=<id>`
- Детальная: `http://localhost:8080/detail/<service_id>?requestId=<id>`
- Корзина/расчёт: `http://localhost:8080/calc?requestId=<id>`
- ORM поиск: `http://localhost:8080/orm/services?q=&thickness=`
- Текущая заявка (JSON): `http://localhost:8080/orm/request/current?userId=1`
- Удаление заявки (RAW): POST `/clear` (форма на `/calc`).
- Проверка ограничений (debug): `http://localhost:8080/debug/requests/constraints`
