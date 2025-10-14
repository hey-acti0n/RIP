#!/bin/bash

echo "🔍 Содержимое Redis для системы UltraRezina"
echo "=============================================="
echo

# Подключение к Redis
REDIS_CMD="docker exec rip-redis-1 redis-cli -a password"

echo "📋 Все ключи в Redis:"
echo "---------------------"
$REDIS_CMD KEYS "*"
echo

echo "🔐 Черный список JWT токенов:"
echo "-----------------------------"
for key in $($REDIS_CMD KEYS "blacklist:*"); do
    echo "Ключ: $key"
    echo "Значение: $($REDIS_CMD GET "$key")"
    echo "TTL (секунды): $($REDIS_CMD TTL "$key")"
    echo "TTL (часы): $(($($REDIS_CMD TTL "$key") / 3600))"
    echo
    
    # Извлекаем JWT payload (средняя часть токена)
    jwt_payload=$(echo "$key" | sed 's/blacklist://' | cut -d'.' -f2)
    
    # Декодируем base64
    echo "👤 Информация о пользователе:"
    echo "$jwt_payload" | base64 -d 2>/dev/null | jq . 2>/dev/null || echo "$jwt_payload" | base64 -d
    echo
    echo "---"
done

echo "📊 Статистика Redis:"
echo "-------------------"
echo "Общее количество ключей: $($REDIS_CMD DBSIZE)"
echo "Использование памяти: $($REDIS_CMD INFO memory | grep used_memory_human | cut -d: -f2)"
