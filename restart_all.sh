#!/bin/bash

echo "🔄 Перезапуск всего приложения..."
echo ""

# Остановка существующих процессов
echo "1. Остановка существующих процессов..."
cd "$(dirname "$0")"
docker-compose down 2>/dev/null
pkill -f "go run main.go" 2>/dev/null
sleep 2

# Ожидание готовности Docker
echo "2. Ожидание готовности Docker..."
for i in {1..30}; do
    if docker ps >/dev/null 2>&1; then
        echo "   ✓ Docker готов!"
        break
    fi
    echo "   Ожидание... ($i/30)"
    sleep 2
done

if ! docker ps >/dev/null 2>&1; then
    echo "   ✗ Docker daemon не запущен. Пожалуйста, запустите Docker Desktop вручную."
    exit 1
fi

# Освобождение портов
echo "3. Освобождение портов..."
lsof -ti:6379 | xargs kill -9 2>/dev/null
sleep 1

# Запуск Docker контейнеров
echo "4. Запуск Docker контейнеров..."
docker-compose up -d

# Ожидание готовности БД
echo "5. Ожидание готовности базы данных..."
sleep 5

# Настройка MinIO
echo "6. Настройка MinIO..."
mc alias set localminio http://localhost:9000 root rootpassword123 2>/dev/null
mc anonymous set download localminio/images 2>/dev/null

# Запуск backend
echo "7. Запуск backend приложения..."
pkill -f "go run main.go" 2>/dev/null
sleep 1
export USE_HTTPS=true
export SSL_CERT=ssl/cert.pem
export SSL_KEY=ssl/key.pem
nohup go run main.go > server.log 2>&1 &
echo $! > server.pid
sleep 3

# Проверка статуса
echo ""
echo "=== Статус ==="
echo ""
echo "Docker контейнеры:"
docker-compose ps
echo ""
echo "Backend приложение:"
if ps -p $(cat server.pid 2>/dev/null) > /dev/null 2>&1; then
    echo "   ✓ Запущен (PID: $(cat server.pid))"
    echo "   Лог: tail -f server.log"
else
    echo "   ✗ Не запущен. Проверьте server.log"
fi
echo ""
echo "Проверка API:"
curl -k -s 'https://localhost:8080/api/v1/materials?limit=1' >/dev/null 2>&1 && echo "   ✓ API доступен (HTTPS)" || echo "   ✗ API недоступен"
echo ""
echo "Готово! 🎉"


