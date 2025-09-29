package main

import (
	"log"
	"net/http"

	"rip/internal/api/rest"
	"rip/internal/app/repository"
	"rip/internal/app/service"
)

func main() {
	// Инициализируем базу данных
	db := repository.InitDB()

	// Создаем сервис
	svc := service.NewService(db)

	// Создаем роутер
	router := rest.NewRouter(svc)

	// Настраиваем маршруты
	httpRouter := router.SetupRoutes()

	// Запускаем сервер
	addr := repository.Getenv("ADDR", ":8080")
	log.Printf("REST API server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, httpRouter))
}
