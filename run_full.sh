#!/bin/bash

echo "🚀 Запуск полного стека UltraRezina..."

# Создаем сеть если не существует
docker network create app-network 2>/dev/null || true

# Запускаем все сервисы
echo "📦 Запуск сервисов..."
docker-compose -f docker-compose.full.yml up --build -d

echo "⏳ Ожидание запуска сервисов..."
sleep 10

echo "✅ Сервисы запущены:"
echo "   🌐 Frontend: http://localhost:3000"
echo "   🔧 Backend API: http://localhost:8080"
echo "   📊 Swagger UI: http://localhost:8080/swagger/"
echo "   🗄️  PostgreSQL: localhost:5432"
echo "   🔴 Redis: localhost:6379"
echo "   📁 MinIO: http://localhost:9001"

echo ""
echo "📋 Полезные команды:"
echo "   Остановить: docker-compose -f docker-compose.full.yml down"
echo "   Логи: docker-compose -f docker-compose.full.yml logs -f"
echo "   Перезапуск: docker-compose -f docker-compose.full.yml restart"
