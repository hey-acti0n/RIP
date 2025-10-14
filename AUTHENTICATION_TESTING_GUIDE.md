# 🔐 Гайд по тестированию системы аутентификации и авторизации

## 📋 Обзор системы

Система UltraRezina реализует JWT-аутентификацию с ролевой моделью доступа:
- **Гость** - только чтение данных
- **Пользователь** - доступ к своим расчетам
- **Модератор** - доступ ко всем расчетам и методам управления

## 🚀 Подготовка к тестированию

### 1. Запуск сервера
```bash
cd /Users/acti0n/Documents/RIP
./unified_server
```

### 2. Проверка доступности
- **API**: http://localhost:8080/api/v1/
- **Swagger UI**: http://localhost:8080/swagger/index.html

## 📝 Пошаговое тестирование

### Этап 1: Тестирование через Swagger UI

#### 1.1 Аутентификация в режиме инкогнито

1. **Откройте браузер в режиме инкогнито**
2. **Перейдите на**: http://localhost:8080/swagger/index.html
3. **Найдите раздел "Пользователи"**
4. **Выполните регистрацию**:
   - Endpoint: `POST /users/register`
   - Body:
     ```json
     {
       "username": "testuser",
       "email": "test@example.com",
       "password": "password123",
       "full_name": "Test User"
     }
     ```
   - Ожидаемый результат: `201 Created` с данными пользователя

5. **Выполните аутентификацию**:
   - Endpoint: `POST /users/login`
   - Body:
     ```json
     {
       "username": "testuser",
       "password": "password123"
     }
     ```
   - **ВАЖНО**: Скопируйте JWT токен из ответа!

#### 1.2 Авторизация в Swagger

1. **В правом верхнем углу Swagger UI нажмите кнопку "Authorize" 🔒**
2. **В поле "Value" введите**: `Bearer YOUR_JWT_TOKEN`
   - Замените `YOUR_JWT_TOKEN` на токен, полученный на шаге 1.1
3. **Нажмите "Authorize"**
4. **Нажмите "Close"**

**ВАЖНО:** Теперь все защищенные эндпоинты (включая добавление в корзину) будут использовать авторизацию!

#### 1.3 Получение списка расчетов

1. **В том же окне Swagger**:
   - Endpoint: `GET /calculations`
   - Ожидаемый результат: `200 OK` с пустым списком расчетов

#### 1.4 Тестирование корзины

1. **Добавление товара в корзину**:
   - Endpoint: `POST /materials/{id}/add-to-cart`
   - **ВАЖНО:** Теперь этот эндпоинт требует авторизации (есть значок 🔒)
   - Введите ID материала (например, 19)
   - Ожидаемый результат: `200 OK` с `calculation_id` для вашего пользователя

2. **Просмотр корзины**:
   - Endpoint: `GET /calculations/cart-info`
   - Ожидаемый результат: `200 OK` с информацией о корзине

### Этап 2: Тестирование через Postman

#### 2.1 Настройка Postman

1. **Импортируйте коллекцию**: `UltraRezina_API_Collection_Fixed.postman_collection.json`
2. **Установите переменные**:
   - `base_url`: `http://localhost:8080`
   - `jwt_token`: [токен из Swagger]

#### 2.2 Тестирование доступа к расчетам

##### Тест 1: Гость (без авторизации)
```http
GET http://localhost:8080/api/v1/calculations
```
**Ожидаемый результат**: `200 OK` (доступ разрешен для чтения)

##### Тест 2: Пользователь (с авторизацией)
```http
GET http://localhost:8080/api/v1/calculations
Authorization: Bearer [JWT_TOKEN]
```
**Ожидаемый результат**: `200 OK` с расчетами только этого пользователя

##### Тест 3: Создание модератора
```bash
# В терминале выполните:
docker exec rip-db-1 psql -U root -d RIP -c "UPDATE users SET role = 1 WHERE username = 'testuser';"
```

##### Тест 4: Аутентификация модератора
```http
POST http://localhost:8080/api/v1/users/login
Content-Type: application/json

{
  "username": "testuser",
  "password": "password123"
}
```
**Скопируйте новый JWT токен модератора!**

##### Тест 5: Модератор видит все расчеты
```http
GET http://localhost:8080/api/v1/calculations
Authorization: Bearer [MODERATOR_JWT_TOKEN]
```
**Ожидаемый результат**: `200 OK` со всеми расчетами

#### 2.3 Тестирование завершения расчетов

##### Тест 6: Формирование расчета (с обязательными параметрами)
```http
PUT http://localhost:8080/api/v1/calculations/1/form
Authorization: Bearer [USER_JWT_TOKEN]
Content-Type: application/json

{
  "installation_weight": 1000.0,
  "natural_frequency": 50.0
}
```
**Ожидаемый результат**: `200 OK` с результатами расчетов для каждого материала

##### Тест 7: Обычный пользователь пытается завершить расчет
```http
PUT http://localhost:8080/api/v1/calculations/1/status?action=complete
Authorization: Bearer [USER_JWT_TOKEN]
```
**Ожидаемый результат**: `403 Forbidden` - "Insufficient permissions"

##### Тест 8: Модератор завершает расчет
```http
PUT http://localhost:8080/api/v1/calculations/1/status?action=complete
Authorization: Bearer [MODERATOR_JWT_TOKEN]
```
**Ожидаемый результат**: `200 OK` с результатами вычислений

### Этап 3: Проверка Redis

#### 3.1 Просмотр содержимого Redis
```bash
# Подключение к Redis
docker exec rip-redis-1 redis-cli -a password

# Просмотр всех ключей
KEYS *

# Просмотр содержимого (если есть ключи)
GET [ключ]
```

#### 3.2 Тестирование logout
```http
POST http://localhost:8080/api/v1/users/logout
Authorization: Bearer [JWT_TOKEN]
```

**После logout проверьте Redis**:
```bash
docker exec rip-redis-1 redis-cli -a password
KEYS jwt_blacklist:*
```

## 🔍 Ожидаемые результаты

### Статус-коды
- `200 OK` - Успешный запрос
- `201 Created` - Ресурс создан
- `400 Bad Request` - Неверные данные
- `401 Unauthorized` - Не аутентифицирован
- `403 Forbidden` - Недостаточно прав
- `404 Not Found` - Ресурс не найден
- `500 Internal Server Error` - Ошибка сервера

### Права доступа
| Роль | GET /calculations | PUT /calculations/{id}/form | PUT /calculations/{id}/status | DELETE /calculations/{id} |
|------|-------------------|----------------------------|-------------------------------|---------------------------|
| Гость | ✅ (все расчеты) | ❌ | ❌ | ❌ |
| Пользователь | ✅ (только свои) | ✅ (формирование) | ❌ | ❌ |
| Модератор | ✅ (все расчеты) | ✅ (формирование) | ✅ (завершение) | ✅ |

## 🛠️ Дополнительные команды

### Создание тестовых данных
```bash
# Создание расчета
curl -X POST http://localhost:8080/api/v1/materials/1/add-to-cart

# Формирование расчета (с обязательными параметрами)
curl -X PUT http://localhost:8080/api/v1/calculations/1/form \
  -H "Authorization: Bearer [JWT_TOKEN]" \
  -H "Content-Type: application/json" \
  -d '{
    "installation_weight": 1000.0,
    "natural_frequency": 50.0
  }'
```

### Просмотр логов сервера
```bash
# В терминале где запущен сервер
tail -f server.log
```

### Сброс базы данных
```bash
# Остановка сервера
pkill -f unified_server

# Пересоздание базы
docker exec rip-db-1 psql -U root -d RIP -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# Запуск сервера
./unified_server
```

## 📊 Структура JWT токена

JWT токен содержит:
```json
{
  "user_id": 1,
  "username": "testuser",
  "role": 0,
  "iss": "rip-api",
  "sub": "1",
  "exp": 1760475779,
  "nbf": 1760389379,
  "iat": 1760389379
}
```

Где `role`:
- `0` - Пользователь
- `1` - Модератор  
- `2` - Администратор

## 🚨 Устранение неполадок

### Проблема: "Token is blacklisted"
**Решение**: Токен был добавлен в черный список через logout. Получите новый токен через login.

### Проблема: "Insufficient permissions"
**Решение**: Пользователь не имеет необходимых прав. Проверьте роль пользователя в базе данных.

### Проблема: "Database connection failed"
**Решение**: Убедитесь, что Docker контейнеры запущены:
```bash
docker ps
docker compose up -d
```

## 📈 Мониторинг

### Проверка состояния сервера
```bash
curl http://localhost:8080/api/v1/calculations/cart-info
```

### Проверка Redis
```bash
docker exec rip-redis-1 redis-cli -a password ping
```

### Проверка базы данных
```bash
docker exec rip-db-1 psql -U root -d RIP -c "SELECT COUNT(*) FROM users;"
```

---

## ✅ Чек-лист тестирования

- [ ] Swagger UI открывается в режиме инкогнито
- [ ] Регистрация пользователя работает
- [ ] Аутентификация возвращает JWT токен
- [ ] Авторизация в Swagger UI работает (кнопка Authorize)
- [ ] GET /calculations работает без авторизации
- [ ] GET /calculations с авторизацией показывает только расчеты пользователя
- [ ] POST /materials/{id}/add-to-cart требует авторизации (значок 🔒)
- [ ] Добавление товара в корзину работает с авторизацией
- [ ] GET /calculations/cart-info показывает правильную корзину пользователя
- [ ] Создание модератора в базе данных
- [ ] Модератор видит все расчеты
- [ ] Пользователь может формировать расчет (с обязательными параметрами: вес установки и собственная частота)
- [ ] Обычный пользователь получает 403 при попытке завершить расчет
- [ ] Модератор может завершить расчет
- [ ] Redis содержит информацию о сессиях
- [ ] Logout добавляет токен в черный список

**🎉 Поздравляем! Система аутентификации и авторизации работает корректно!**
