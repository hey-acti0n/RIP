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

	// Создаем сервисы
	svc := service.NewService(db)

	// Создаем роутер
	router := rest.NewRouter(svc)
	routes := router.SetupRoutes()

	// Запускаем сервер
	addr := repository.Getenv("ADDR", ":8080")
	log.Printf("UltraRezina REST API server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, routes))
}
