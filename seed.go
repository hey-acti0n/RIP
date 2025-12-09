package main

import (
	"log"
	"rip/internal/app/repository"
)

func main() {
	log.Println("Заполнение базы данных тестовыми данными...")

	// Инициализируем подключение к БД
	db := repository.InitDB()

	// Создаем пользователей
	users := []repository.User{
		{
			Username: "admin",
			Email:    "admin@ultrarezina.ru",
			Password: "admin123", // Пароль хранится без хеширования (как реализовано в HashPassword)
			FullName: "Администратор Системы",
			Role:     2, // admin
			IsActive: true,
		},
		{
			Username: "moderator",
			Email:    "moderator@ultrarezina.ru",
			Password: "moderator123",
			FullName: "Модератор Системы",
			Role:     1, // moderator
			IsActive: true,
		},
		{
			Username: "user1",
			Email:    "user1@ultrarezina.ru",
			Password: "user123",
			FullName: "Иван Петров",
			Role:     0, // user
			IsActive: true,
		},
	}

	for _, user := range users {
		var existingUser repository.User
		if err := db.Where("username = ? OR email = ?", user.Username, user.Email).First(&existingUser).Error; err == nil {
			log.Printf("Пользователь %s уже существует, пропускаем", user.Username)
			continue
		}
		if err := db.Create(&user).Error; err != nil {
			log.Printf("Ошибка при создании пользователя %s: %v", user.Username, err)
		} else {
			log.Printf("Создан пользователь: %s (%s)", user.Username, user.Email)
		}
	}

	// Создаем материалы
	// В image_url храним путь относительно /images/ в MinIO
	// Код автоматически добавит http://localhost:9000/images/ если путь не начинается с http
	materials := []repository.DBMaterial{
		{
			Name:        "Виброизоляционная прокладка EPDM",
			Description: "Высококачественная виброизоляционная прокладка из этилен-пропилен-диенового каучука (EPDM). Обеспечивает эффективную виброизоляцию промышленного оборудования.",
			IsActive:    true,
			ImageURL:    "/images/epdm_pad.jpg", // Путь будет преобразован в http://localhost:9000/images/epdm_pad.jpg
			Density:     floatPtr(120.5),
			Thickness:   floatPtr(10.0),
			Material:    stringPtr("EPDM"),
		},
		{
			Name:        "Резиновая прокладка NBR",
			Description: "Прокладка из нитрил-бутадиенового каучука (NBR) с отличной маслостойкостью. Идеально подходит для применения в условиях воздействия масел и топлива.",
			IsActive:    true,
			ImageURL:    "/images/nbr_pad.jpg",
			Density:     floatPtr(95.0),
			Thickness:   floatPtr(8.0),
			Material:    stringPtr("NBR"),
		},
		{
			Name:        "Виброизолятор неопреновый",
			Description: "Неопреновая виброизоляционная прокладка с высокой эластичностью. Обеспечивает надежную защиту от вибраций и ударов.",
			IsActive:    true,
			ImageURL:    "/images/neoprene_vibro.jpg",
			Density:     floatPtr(110.0),
			Thickness:   floatPtr(12.0),
			Material:    stringPtr("Неопрен"),
		},
		{
			Name:        "Резиновая прокладка SBR",
			Description: "Стирол-бутадиеновая резиновая прокладка с хорошими механическими свойствами. Широко применяется в автомобильной промышленности.",
			IsActive:    true,
			ImageURL:    "/images/sbr_pad.jpg",
			Density:     floatPtr(88.5),
			Thickness:   floatPtr(6.0),
			Material:    stringPtr("SBR"),
		},
		{
			Name:        "Виброизоляционная прокладка силиконовая",
			Description: "Силиконовая прокладка с отличной термостойкостью и устойчивостью к старению. Работает в широком диапазоне температур от -60°C до +200°C.",
			IsActive:    true,
			ImageURL:    "/images/silicone_pad.jpg",
			Density:     floatPtr(105.0),
			Thickness:   floatPtr(15.0),
			Material:    stringPtr("Силикон"),
		},
		{
			Name:        "Резиновая прокладка FKM (Viton)",
			Description: "Фторкаучуковая прокладка с исключительной химической стойкостью. Подходит для агрессивных сред и высоких температур.",
			IsActive:    true,
			ImageURL:    "/images/fkm_pad.jpg",
			Density:     floatPtr(180.0),
			Thickness:   floatPtr(5.0),
			Material:    stringPtr("FKM"),
		},
		{
			Name:        "Виброизолятор комбинированный",
			Description: "Комбинированная виброизоляционная прокладка из нескольких слоев различных материалов. Обеспечивает оптимальную виброизоляцию.",
			IsActive:    true,
			ImageURL:    "/images/combined_vibro.jpg",
			Density:     floatPtr(130.0),
			Thickness:   floatPtr(20.0),
			Material:    stringPtr("Комбинированный"),
		},
		{
			Name:        "Резиновая прокладка натуральная",
			Description: "Прокладка из натурального каучука с высокой эластичностью и прочностью. Экологически чистый материал.",
			IsActive:    true,
			ImageURL:    "/images/natural_rubber.jpg",
			Density:     floatPtr(92.0),
			Thickness:   floatPtr(10.0),
			Material:    stringPtr("Натуральный каучук"),
		},
	}

	for _, material := range materials {
		var existingMaterial repository.DBMaterial
		if err := db.Where("name = ?", material.Name).First(&existingMaterial).Error; err == nil {
			log.Printf("Материал %s уже существует, пропускаем", material.Name)
			continue
		}
		if err := db.Create(&material).Error; err != nil {
			log.Printf("Ошибка при создании материала %s: %v", material.Name, err)
		} else {
			log.Printf("Создан материал: %s", material.Name)
		}
	}

	log.Println("Заполнение базы данных завершено!")
}

// Вспомогательные функции для создания указателей
func floatPtr(f float64) *float64 {
	return &f
}

func stringPtr(s string) *string {
	return &s
}

