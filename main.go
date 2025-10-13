// @title UltraRezina API
// @version 1.0
// @description API для системы расчета резинотехнических изделий
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT токен для аутентификации. Используйте формат: "Bearer <token>"

package main

import (
	"log"
	"net/http"

	"rip/docs"
	"rip/internal/api/rest"
	"rip/internal/app/redis"
	"rip/internal/app/repository"
	"rip/internal/app/service"
)

func main() {
	// Инициализируем Swagger
	docs.SwaggerInfo.Title = "UltraRezina API"
	docs.SwaggerInfo.Description = "API для системы расчета резинотехнических изделий"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:8080"
	docs.SwaggerInfo.BasePath = "/api/v1"

	// Инициализируем базу данных
	db := repository.InitDB()

	// Создаем сервисы
	svc := service.NewService(db)

	// Инициализируем Redis
	redisClient, err := redis.NewClient("localhost:6379", "password", 0)
	if err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
		log.Println("Continuing without Redis...")
	} else {
		svc.SetRedisClient(redisClient)
		log.Println("Redis connected successfully")
	}

	// Создаем роутер
	router := rest.NewRouter(svc)
	routes := router.SetupRoutes()

	// Запускаем сервер
	addr := repository.Getenv("ADDR", ":8080")
	log.Printf("UltraRezina REST API server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, routes))
}
