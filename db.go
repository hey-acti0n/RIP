package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- DB models ---

type DBService struct {
	ID          int        `gorm:"primaryKey;column:id" json:"id"`
	Name        string     `gorm:"size:255;not null;column:name" json:"name"`
	Description string     `gorm:"type:text;column:description" json:"description"`
	IsActive    bool       `gorm:"column:is_active;default:true" json:"is_active"`
	ImageURL    string     `gorm:"type:text;column:image_url" json:"image_url"`
	Density     *float64   `gorm:"column:density" json:"density,omitempty"`
	Thickness   *float64   `gorm:"column:thickness" json:"thickness,omitempty"`
	Material    *string    `gorm:"size:50;column:material" json:"material,omitempty"`
	CreatedAt   *time.Time `gorm:"autoCreateTime;column:created_at" json:"created_at,omitempty"`
}

func (DBService) TableName() string { return "services" }

type Request struct {
	ID          int        `gorm:"primaryKey;column:id" json:"id"`
	Status      string     `gorm:"size:50;column:status" json:"status"`
	CreatedAt   time.Time  `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	CreatorID   int        `gorm:"column:creator_id" json:"creator_id"`
	FormedAt    *time.Time `gorm:"column:formed_at" json:"formed_at,omitempty"`
	CompletedAt *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
	ModeratorID *int       `gorm:"column:moderator_id" json:"moderator_id,omitempty"`
}

func (Request) TableName() string { return "requests" }

type RequestService struct {
	RequestID int       `gorm:"primaryKey;column:request_id" json:"request_id"`
	ServiceID int       `gorm:"primaryKey;column:service_id" json:"service_id"`
	Quantity  int       `gorm:"not null;default:1;column:quantity" json:"quantity"`
	SortOrder int       `gorm:"not null;default:0;column:sort_order" json:"sort_order"`
	IsMain    bool      `gorm:"not null;default:false;column:is_main" json:"is_main"`
	CreatedAt time.Time `gorm:"autoCreateTime;column:created_at" json:"created_at"`
	Service   DBService `gorm:"foreignKey:ServiceID" json:"-"`
}

func (RequestService) TableName() string { return "request_services" }

// Getenv helper function
func Getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// WriteJSON helper function
func WriteJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// InitDB establishes connection using env DATABASE_DSN or docker-compose defaults
func InitDB() *gorm.DB {
	dsn := Getenv("DATABASE_DSN", "host=localhost user=postgres password=root dbname=rip port=5432 sslmode=disable TimeZone=UTC")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	// AutoMigrate to ensure tables exist (safe for demo)
	_ = db.AutoMigrate(&DBService{}, &Request{}, &RequestService{})
	return db
}

// MountORMRoutes registers endpoints required by assignment
func MountORMRoutes(db *gorm.DB) {
	// 1) GET services search via ORM
	http.HandleFunc("/orm/services", func(w http.ResponseWriter, r *http.Request) {
		q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
		thickness := strings.TrimSpace(r.URL.Query().Get("thickness"))
		tx := db.Model(&DBService{}).Where("is_active = ?", true)
		if q != "" {
			tx = tx.Where("LOWER(name) LIKE ?", "%"+q+"%")
		}
		if thickness != "" {
			tx = tx.Where("CAST(thickness AS TEXT) LIKE ?", "%"+thickness+"%")
		}
		var items []DBService
		if err := tx.Order("id").Find(&items).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Формируем полные URL для MinIO
		minioHost := Getenv("MINIO_PUBLIC_ENDPOINT", "http://localhost:9000")
		assetsBase := fmt.Sprintf("%s/images", minioHost)
		var itemsWithURLs []map[string]any
		for _, item := range items {
			itemMap := map[string]any{
				"id":          item.ID,
				"name":        item.Name,
				"description": item.Description,
				"is_active":   item.IsActive,
				"image_url":   item.ImageURL,
				"density":     item.Density,
				"thickness":   item.Thickness,
				"material":    item.Material,
				"created_at":  item.CreatedAt,
			}
			// Если image_url не полный URL, добавляем базовый URL MinIO
			if !strings.HasPrefix(item.ImageURL, "http") {
				// Если URL уже начинается с /images/, добавляем только базовый хост
				if strings.HasPrefix(item.ImageURL, "/images/") {
					itemMap["image_url"] = "http://localhost:9000" + item.ImageURL
				} else {
					itemMap["image_url"] = assetsBase + item.ImageURL
				}
			}
			itemsWithURLs = append(itemsWithURLs, itemMap)
		}
		WriteJSON(w, map[string]any{"count": len(items), "items": itemsWithURLs})
	})

	// 2) POST add service to current draft request via ORM
	http.HandleFunc("/orm/request/add", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userID, _ := strconv.Atoi(r.FormValue("userId"))
		if userID == 0 {
			userID = 1
		}
		serviceID, _ := strconv.Atoi(r.FormValue("serviceId"))
		if serviceID == 0 {
			http.Error(w, "serviceId required", http.StatusBadRequest)
			return
		}

		// find or create draft request (используем default БД для status)
		var req Request
		if err := db.Where("creator_id = ? AND status = ?", userID, "pending").First(&req).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Model(&Request{}).Create(map[string]any{"creator_id": userID, "status": "pending"}).Error; err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				_ = db.Where("creator_id = ?", userID).Order("id desc").First(&req).Error
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		rs := RequestService{RequestID: req.ID, ServiceID: serviceID, Quantity: 1}
		// upsert by composite PK
		if err := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "request_id"}, {Name: "service_id"}}, DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("request_services.quantity + 1")})}).Create(&rs).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		WriteJSON(w, map[string]any{"requestId": req.ID, "success": true})
	})

	// 3) GET current draft request via ORM
	http.HandleFunc("/orm/request/current", func(w http.ResponseWriter, r *http.Request) {
		userID, _ := strconv.Atoi(r.URL.Query().Get("userId"))
		if userID == 0 {
			userID = 1
		}
		var req Request
		if err := db.Where("creator_id = ? AND status = ?", userID, "pending").First(&req).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				WriteJSON(w, map[string]any{"exists": false})
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var items []RequestService
		_ = db.Preload("Service").Where("request_id = ?", req.ID).Order("sort_order, service_id").Find(&items).Error
		WriteJSON(w, map[string]any{"exists": true, "request": req, "items": items})
	})

	// 4) POST logical delete via raw SQL (no ORM)
	http.HandleFunc("/orm/request/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.FormValue("requestId")
		if id == "" {
			http.Error(w, "requestId required", http.StatusBadRequest)
			return
		}
		if err := db.Exec("UPDATE requests SET status = 'rejected' WHERE id = ?", id).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		WriteJSON(w, map[string]any{"deleted": true})
	})

	// 5) GET request by id: do not allow viewing deleted
	http.HandleFunc("/orm/request", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.NotFound(w, r)
			return
		}
		var req Request
		if err := db.Where("id = ? AND status <> 'rejected'", id).First(&req).Error; err != nil {
			http.Error(w, "request not found or deleted", http.StatusNotFound)
			return
		}
		WriteJSON(w, req)
	})

	// DEBUG: constraints of requests table
	http.HandleFunc("/debug/requests/constraints", func(w http.ResponseWriter, r *http.Request) {
		type Row struct{ Def string }
		var rows []Row
		q := `SELECT pg_get_constraintdef(c.oid) as def
			FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid
			WHERE t.relname='requests' AND contype='c'` // only CHECK
		if err := db.Raw(q).Scan(&rows).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		WriteJSON(w, rows)
	})
}
