package app

import (
	"fmt"
	"os"
)

type Store struct {
	Services []Service
	Carts    map[string][]CartItem
}

func NewStore() *Store {
	minioHost := getenv("MINIO_PUBLIC_ENDPOINT", "http://localhost:9000")
	mk := func(name, obj string) Service {
		var props []string
		var description string
		switch name {
		case "ARP-200":
			description = "Акустическая резиновая плита для виброизоляции"
			props = []string{"Плотность: 200 кг/м³", "Толщина: 8-20 мм", "Материал: EPDM резина"}
		case "CorkMat-8":
			description = "Пробковый виброизоляционный материал"
			props = []string{"Плотность: 120 кг/м³", "Толщина: 8 мм", "Материал: Пробка"}
		case "Fiberglass Vib":
			description = "Стекловолоконный виброизоляционный материал"
			props = []string{"Плотность: 80 кг/м³", "Толщина: 10-25 мм", "Материал: Стекловолокно"}
		case "HomePro":
			description = "Профессиональный виброизоляционный материал для дома"
			props = []string{"Плотность: 150 кг/м³", "Толщина: 12 мм", "Материал: Полиуретан"}
		case "Multivib":
			description = "Универсальный многослойный виброизоляционный материал"
			props = []string{"Плотность: 180 кг/м³", "Толщина: 15 мм", "Материал: Композит"}
		case "N-300":
			description = "Высокоплотный виброизоляционный материал"
			props = []string{"Плотность: 300 кг/м³", "Толщина: 6-12 мм", "Материал: Неопрен"}
		case "PU-450":
			description = "Полиуретановый виброизоляционный материал повышенной плотности"
			props = []string{"Плотность: 450 кг/м³", "Толщина: 8-16 мм", "Материал: Полиуретан"}
		case "SpringMaster":
			description = "Пружинная виброизоляционная система"
			props = []string{"Плотность: 200 кг/м³", "Толщина: 20 мм", "Материал: Сталь + резина"}
		case "VibroFoam":
			description = "Пенополиуретановый виброизоляционный материал"
			props = []string{"Плотность: 60 кг/м³", "Толщина: 10-30 мм", "Материал: Пенополиуретан"}
		case "VS-1000":
			description = "Высокоэффективный виброизоляционный материал"
			props = []string{"Плотность: 1000 кг/м³", "Толщина: 5-10 мм", "Материал: Свинцовая резина"}
		default:
			description = "Виброизоляционный материал для промышленного оборудования"
			props = []string{"Плотность: 120 кг/м³", "Толщина: 10 мм", "Материал: EPDM"}
		}
		return Service{
			ID:          len(name) + len(obj),
			Name:        name,
			Description: description,
			ImageURL:    fmt.Sprintf("%s/%s/%s", minioHost, "images", obj),
			Props:       props,
		}
	}
	return &Store{
		Services: []Service{
			mk("ARP-200", "arp-200.jpg"),
			mk("CorkMat-8", "corkmat-8.jpg"),
			mk("Fiberglass Vib", "fiberglass-vib.jpg"),
			mk("HomePro", "homepro.jpg"),
			mk("Multivib", "multivib.jpg"),
			mk("N-300", "n-300.jpg"),
			mk("PU-450", "pu-450.jpg"),
			mk("SpringMaster", "springmaster.jpg"),
			mk("VibroFoam", "vibrofoam.jpg"),
			mk("VS-1000", "vs-1000.jpg"),
		},
		Carts: map[string][]CartItem{},
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
