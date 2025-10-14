package rest

import (
	"net/http"
	"time"

	"rip/internal/app/middleware"
	"rip/internal/app/models"
	"rip/internal/app/redis"
	"rip/internal/app/service"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

// Router настраивает маршруты REST API
type Router struct {
	service        *service.Service
	authMiddleware *middleware.AuthMiddleware
}

// NewRouter создает новый роутер
func NewRouter(svc *service.Service) *Router {
	// Создаем JWT сервис
	jwtService := service.NewJWTService("your-secret-key", 24*time.Hour, 7*24*time.Hour)

	// Создаем Redis клиент
	redisClient, err := redis.NewClient("localhost:6379", "password", 0)
	if err != nil {
		// Если Redis недоступен, продолжаем без него
		redisClient = nil
	}

	// Устанавливаем Redis клиент в сервис
	svc.SetRedisClient(redisClient)

	// Создаем middleware аутентификации
	authMiddleware := middleware.NewAuthMiddleware(jwtService, redisClient)

	return &Router{
		service:        svc,
		authMiddleware: authMiddleware,
	}
}

// SetupRoutes настраивает все маршруты API
func (r *Router) SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Создаем обработчики
	materialHandler := NewMaterialHandler(r.service)
	calculationHandler := NewCalculationHandler(r.service)
	materialCalculationHandler := NewMaterialCalculationHandler(r.service)
	userHandler := NewUserHandler(r.service)

	// Swagger документация
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// API v1
	apiV1 := router.PathPrefix("/api/v1").Subrouter()

	// Домен материала
	materials := apiV1.PathPrefix("/materials").Subrouter()
	materials.HandleFunc("", r.authMiddleware.OptionalAuth(materialHandler.GetMaterials)).Methods("GET")
	materials.HandleFunc("/{id}", r.authMiddleware.OptionalAuth(materialHandler.GetMaterial)).Methods("GET")
	materials.HandleFunc("", r.authMiddleware.RequireAuth(r.authMiddleware.RequireRole(models.ModeratorRole)(materialHandler.CreateMaterial))).Methods("POST")
	materials.HandleFunc("/{id}", r.authMiddleware.RequireAuth(r.authMiddleware.RequireRole(models.ModeratorRole)(materialHandler.UpdateMaterial))).Methods("PUT")
	materials.HandleFunc("/{id}", r.authMiddleware.RequireAuth(r.authMiddleware.RequireRole(models.ModeratorRole)(materialHandler.DeleteMaterial))).Methods("DELETE")
	materials.HandleFunc("/{id}/add-to-cart", r.authMiddleware.OptionalAuth(materialHandler.AddMaterialToCart)).Methods("POST")
	materials.HandleFunc("/{id}/image", r.authMiddleware.RequireAuth(r.authMiddleware.RequireRole(models.ModeratorRole)(materialHandler.UploadMaterialImage))).Methods("POST")

	// Домен расчёта
	calculations := apiV1.PathPrefix("/calculations").Subrouter()
	calculations.HandleFunc("/cart-info", r.authMiddleware.OptionalAuth(calculationHandler.GetCartInfo)).Methods("GET")
	calculations.HandleFunc("", r.authMiddleware.OptionalAuth(calculationHandler.GetCalculations)).Methods("GET")
	calculations.HandleFunc("/{id}", r.authMiddleware.OptionalAuth(calculationHandler.GetCalculation)).Methods("GET")
	calculations.HandleFunc("/{id}/form", r.authMiddleware.RequireAuth(calculationHandler.FormCalculation)).Methods("PUT")
	calculations.HandleFunc("/{id}/status", r.authMiddleware.RequireAuth(r.authMiddleware.RequireRole(models.ModeratorRole)(calculationHandler.CompleteCalculation))).Methods("PUT")
	calculations.HandleFunc("/{id}/materials", r.authMiddleware.OptionalAuth(calculationHandler.GetCalculationMaterials)).Methods("GET")
	calculations.HandleFunc("/{id}", r.authMiddleware.RequireAuth(r.authMiddleware.RequireRole(models.ModeratorRole)(calculationHandler.DeleteCalculation))).Methods("DELETE")

	// Домен м-м (расчёт-материал)
	materialCalculations := apiV1.PathPrefix("/calculations/{calculationId}/materials").Subrouter()
	materialCalculations.HandleFunc("/{materialId}", r.authMiddleware.RequireAuth(materialCalculationHandler.DeleteMaterialCalculation)).Methods("DELETE")
	materialCalculations.HandleFunc("/{materialId}", r.authMiddleware.RequireAuth(materialCalculationHandler.UpdateMaterialCalculation)).Methods("PUT")

	// Домен пользователь
	users := apiV1.PathPrefix("/users").Subrouter()
	users.HandleFunc("/register", userHandler.Register).Methods("POST")
	users.HandleFunc("/login", userHandler.Login).Methods("POST")
	users.HandleFunc("/profile", r.authMiddleware.RequireAuth(userHandler.GetProfile)).Methods("GET")
	users.HandleFunc("/profile", r.authMiddleware.RequireAuth(userHandler.UpdateProfile)).Methods("PUT")
	users.HandleFunc("/logout", r.authMiddleware.RequireAuth(userHandler.Logout)).Methods("POST")

	// Middleware для CORS
	router.Use(corsMiddleware)

	return router
}

// corsMiddleware добавляет CORS заголовки
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
