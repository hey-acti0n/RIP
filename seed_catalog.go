package main

import (
	"log"
	"rip/internal/app/repository"
)

func main() {
	log.Println("Добавление материалов из каталога в базу данных...")

	// Инициализируем подключение к БД
	db := repository.InitDB()

	// Создаем материалы на основе загруженных картинок
	materials := []repository.DBMaterial{
		{
			Name:        "Виброизолятор ARP-200",
			Description: "Высокоэффективный виброизолятор ARP-200 для промышленного оборудования. Обеспечивает надежную защиту от вибраций и шума.",
			IsActive:    true,
			ImageURL:    "/images/arp-200.jpg",
			Density:     floatPtr(200.0),
			Thickness:   floatPtr(10.0),
			Material:    stringPtr("Резина"),
		},
		{
			Name:        "Пробковая прокладка 8мм",
			Description: "Пробковая виброизоляционная прокладка толщиной 8мм. Экологически чистый материал с отличными звукоизоляционными свойствами.",
			IsActive:    true,
			ImageURL:    "/images/corkmat-8.jpg",
			Density:     floatPtr(120.0),
			Thickness:   floatPtr(8.0),
			Material:    stringPtr("Пробка"),
		},
		{
			Name:        "Стекловолоконный виброизолятор",
			Description: "Виброизолятор из стекловолокна с высокой прочностью и устойчивостью к температурным воздействиям. Идеален для промышленных применений.",
			IsActive:    true,
			ImageURL:    "/images/fiberglass-vib.jpg",
			Density:     floatPtr(150.0),
			Thickness:   floatPtr(12.0),
			Material:    stringPtr("Стекловолокно"),
		},
		{
			Name:        "Виброизолятор HomePro",
			Description: "Бытовой виброизолятор HomePro для домашнего использования. Легкий монтаж и эффективная защита от вибраций бытовой техники.",
			IsActive:    true,
			ImageURL:    "/images/homepro.jpg",
			Density:     floatPtr(90.0),
			Thickness:   floatPtr(6.0),
			Material:    stringPtr("Полиуретан"),
		},
		{
			Name:        "Многослойный виброизолятор MultiVib",
			Description: "Многослойный виброизолятор MultiVib с комбинированной структурой. Обеспечивает максимальную эффективность виброизоляции.",
			IsActive:    true,
			ImageURL:    "/images/multivib.jpg",
			Density:     floatPtr(130.0),
			Thickness:   floatPtr(15.0),
			Material:    stringPtr("Комбинированный"),
		},
		{
			Name:        "Виброизолятор N-300",
			Description: "Промышленный виброизолятор N-300 с повышенной нагрузочной способностью. Предназначен для тяжелого оборудования.",
			IsActive:    true,
			ImageURL:    "/images/n-300.jpg",
			Density:     floatPtr(300.0),
			Thickness:   floatPtr(20.0),
			Material:    stringPtr("Резина"),
		},
		{
			Name:        "Полиуретановый виброизолятор PU-450",
			Description: "Высоконагруженный полиуретановый виброизолятор PU-450. Отличается высокой прочностью и долговечностью.",
			IsActive:    true,
			ImageURL:    "/images/pu-450.jpg",
			Density:     floatPtr(450.0),
			Thickness:   floatPtr(25.0),
			Material:    stringPtr("Полиуретан"),
		},
		{
			Name:        "Виброизолятор SpringMaster",
			Description: "Пружинный виброизолятор SpringMaster с демпфирующими элементами. Обеспечивает эффективную изоляцию низкочастотных вибраций.",
			IsActive:    true,
			ImageURL:    "/images/springmaster.jpg",
			Density:     floatPtr(180.0),
			Thickness:   floatPtr(30.0),
			Material:    stringPtr("Пружинный"),
		},
		{
			Name:        "Вибропена",
			Description: "Вспененный виброизоляционный материал с закрытыми порами. Легкий и эффективный материал для виброизоляции.",
			IsActive:    true,
			ImageURL:    "/images/vibrofoam.jpg",
			Density:     floatPtr(45.0),
			Thickness:   floatPtr(10.0),
			Material:    stringPtr("Вспененный материал"),
		},
		{
			Name:        "Виброизолятор VS-1000",
			Description: "Мощный виброизолятор VS-1000 для особо тяжелого оборудования. Максимальная нагрузочная способность и эффективность.",
			IsActive:    true,
			ImageURL:    "/images/vs-1000.jpg",
			Density:     floatPtr(1000.0),
			Thickness:   floatPtr(40.0),
			Material:    stringPtr("Резина"),
		},
	}

	addedCount := 0
	skippedCount := 0

	for _, material := range materials {
		var existingMaterial repository.DBMaterial
		if err := db.Where("name = ?", material.Name).First(&existingMaterial).Error; err == nil {
			log.Printf("Материал '%s' уже существует, пропускаем", material.Name)
			skippedCount++
			continue
		}
		if err := db.Create(&material).Error; err != nil {
			log.Printf("Ошибка при создании материала '%s': %v", material.Name, err)
		} else {
			log.Printf("✓ Создан материал: %s (путь: %s)", material.Name, material.ImageURL)
			addedCount++
		}
	}

	log.Printf("\n=== Итоги ===")
	log.Printf("Добавлено новых материалов: %d", addedCount)
	log.Printf("Пропущено (уже существуют): %d", skippedCount)
	log.Printf("Всего обработано: %d", len(materials))
	log.Println("Готово!")
}

// Вспомогательные функции для создания указателей
func floatPtr(f float64) *float64 {
	return &f
}

func stringPtr(s string) *string {
	return &s
}

