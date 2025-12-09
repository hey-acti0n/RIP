#!/bin/bash
echo "🚀 Запуск REST API сервера..."
echo "📍 API будет доступен по адресу: https://localhost:8080/api/v1/"
echo "📋 Используйте Postman коллекцию для тестирования"
echo "⏹️  Для остановки нажмите Ctrl+C"
echo ""

export USE_HTTPS=true
export SSL_CERT=ssl/cert.pem
export SSL_KEY=ssl/key.pem

go run main.go
