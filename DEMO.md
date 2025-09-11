# Демонстрация UltraRezina - Инструкция для преподавателя

## Запуск приложения
```bash
cd /Users/acti0n/Documents/RIP
docker compose up -d  # Поднять MinIO
go run main.go        # Запустить сервер на :8080
```

## План демонстрации

### 1. Показать три страницы приложения с поиском

**Страница 1: Каталог** - `http://localhost:8080/`
- Поиск по названию материала
- Фильтр по толщине
- Список товаров с кнопками "Подробнее" и "Купить"

**Страница 2: Детальная страница** - `http://localhost:8080/detail/{id}`
- Подробная информация о товаре
- Кнопка "Купить"

**Страница 3: Корзина/Расчёт** - `http://localhost:8080/calc`
- Список товаров в корзине
- Форма для ввода параметров расчёта

### 2. Network запросы (3 GET) - показать во вкладке Network

#### Запрос 1: Поиск и фильтрация
```
GET /api/services?q=ultra&thickness=10&requestId=1
```
**Параметры:**
- `q=ultra` - фильтр по названию (поиск)
- `thickness=10` - фильтр по толщине
- `requestId=1` - ID заявки для отслеживания корзины

#### Запрос 2: Добавление в корзину
```
GET /api/add?requestId=1&serviceId=28
```
**Параметры:**
- `requestId=1` - ID заявки
- `serviceId=28` - ID услуги/товара

#### Запрос 3: Получение корзины
```
GET /api/cart?requestId=1
```
**Параметры:**
- `requestId=1` - ID заявки

### 3. HTML из Response - показать в Network

**В ответе на `/api/services` содержится:**
- `requestId` - ID заявки
- `count` - количество найденных товаров
- `items[]` - массив товаров с полями:
  - `id` - ID услуги
  - `name` - название
  - `imageUrl` - URL изображения из MinIO
  - `props[]` - характеристики

**В ответе на `/api/cart` содержится:**
- `requestId` - ID заявки
- `items[]` - товары в корзине с полями:
  - `serviceId` - ID услуги
  - `quantity` - количество

### 4. URL изображений из MinIO в коде

**Коллекция услуг** (`main.go:48-72`):
```go
func newStore() *Store {
    minioHost := getenv("MINIO_PUBLIC_ENDPOINT", "http://localhost:9000")
    mk := func(name, obj string) Service {
        return Service{
            ID:          len(name) + len(obj),
            Name:        name,
            Description: "Виброизоляционный материал для промышленного оборудования",
            ImageURL:    fmt.Sprintf("%s/%s/%s", minioHost, "images", obj),
            Props:       []string{"Плотность: 120 кг/м³", "Толщина: 10 мм", "Материал: EPDM"},
        }
    }
    // ...
}
```

**Использование в 3 шаблонах:**

1. **catalog.html** - логотип и иконки:
```html
<img src="{{.AssetsBase}}/logo.png" alt="UltraRezina" />
<img src="{{.AssetsBase}}/search.png" alt="search" />
<img src="{{.AssetsBase}}/cart.png" alt="Корзина" />
```

2. **detail.html** - изображение товара:
```html
<img class="detail-image" src="{{.Service.ImageURL}}" alt="{{.Service.Name}}" />
```

3. **calc.html** - не используется напрямую, но данные передаются через `{{.AssetsBase}}`

### 5. Роутинг: 3 URL и три контроллера

**Роуты** (`main.go:508-510`):
```go
http.HandleFunc("/", srv.handleCatalog)        // Каталог
http.HandleFunc("/calc", srv.handleCalc)       // Корзина/Расчёт  
http.HandleFunc("/detail/", srv.handleDetail)  // Детальная страница
```

**Контроллеры:**

1. **handleCatalog** (`main.go:119-152`) - каталог с поиском
2. **handleCalc** (`main.go:154-203`) - страница корзины и расчёта
3. **handleDetail** (`main.go:205-226`) - детальная страница товара

### 6. Реализация фильтрации/поиска

**Функция фильтрации** (`main.go:358-380`):
```go
func (s *Server) filterServices(q, thickness string) []Service {
    q = strings.ToLower(strings.TrimSpace(q))
    thickness = strings.TrimSpace(thickness)
    var list []Service
    for _, sv := range s.store.services {
        // Фильтр по названию
        nameMatch := q == "" || strings.Contains(strings.ToLower(sv.Name), q)
        
        // Фильтр по толщине
        thicknessMatch := true
        if thickness != "" {
            thicknessMatch = false
            for _, prop := range sv.Props {
                if strings.Contains(strings.ToLower(prop), "толщина") &&
                    strings.Contains(strings.ToLower(prop), strings.ToLower(thickness)) {
                    thicknessMatch = true
                    break
                }
            }
        }
        
        if nameMatch && thicknessMatch {
            list = append(list, sv)
        }
    }
    return list
}
```

**Использование в каталоге** (`main.go:128`):
```go
services := s.filterServices(q, thickness)
```

## Демо-сценарий для преподавателя

1. **Откройте каталог**: `http://localhost:8080`
   - Показать поиск и фильтр
   - Ввести "ultra" в поиск → показать Network запрос

2. **Добавить товар в корзину**:
   - Нажать "Купить" → показать Network запрос `/api/add`
   - Остаться на каталоге (товар добавлен)

3. **Перейти в корзину**: `http://localhost:8080/calc`
   - Показать список товаров в корзине
   - Показать форму расчёта (без кнопки)

4. **Показать детальную страницу**:
   - Нажать "Подробнее" на товаре
   - Показать изображение товара из MinIO
   - Нажать "Купить" → перейти в корзину

## Соответствие требованиям

✅ **3 страницы**: каталог, детальная, корзина/расчёт  
✅ **Поиск и фильтрация**: по названию и толщине  
✅ **3 GET запроса**: поиск, добавление, корзина  
✅ **ID заявки**: передается во всех запросах  
✅ **URL изображений**: из MinIO в коллекции услуг  
✅ **Роутинг**: 3 URL с соответствующими контроллерами  
✅ **Фильтрация**: реализована в коде с объяснением логики