#!/bin/bash
echo "🚀 Запуск REST API сервера..."
echo "📍 API будет доступен по адресу: http://localhost:8080/api/v1/"
echo "📋 Используйте Postman коллекцию для тестирования"
echo "⏹️  Для остановки нажмите Ctrl+C"
echo ""

go run main.go
