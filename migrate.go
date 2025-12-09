package main

import (
	"log"
	"rip/internal/app/repository"
)

func main() {
	log.Println("Запуск миграций базы данных...")
	db := repository.InitDB()
	if db != nil {
		log.Println("Миграции выполнены успешно!")
	}
}

