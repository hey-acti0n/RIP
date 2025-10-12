package rest

import (
	"net/http"

	"rip/internal/app/service"

	"github.com/gorilla/mux"
)

// Router настраивает маршруты REST API
type Router struct {
	service *service.Service
}

// NewRouter создает новый роутер
func NewRouter(svc *service.Service) *Router {
	return &Router{service: svc}
}

// SetupRoutes настраивает все маршруты API
func (r *Router) SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Создаем обработчики
	materialHandler := NewMaterialHandler(r.service)
	calculationHandler := NewCalculationHandler(r.service)
	materialCalculationHandler := NewMaterialCalculationHandler(r.service)
	userHandler := NewUserHandler(r.service)

	// API v1
	apiV1 := router.PathPrefix("/api/v1").Subrouter()

	// Домен материала
	materials := apiV1.PathPrefix("/materials").Subrouter()
	materials.HandleFunc("", materialHandler.GetMaterials).Methods("GET")
	materials.HandleFunc("/{id}", materialHandler.GetMaterial).Methods("GET")
	materials.HandleFunc("", materialHandler.CreateMaterial).Methods("POST")
	materials.HandleFunc("/{id}", materialHandler.UpdateMaterial).Methods("PUT")
	materials.HandleFunc("/{id}", materialHandler.DeleteMaterial).Methods("DELETE")
	materials.HandleFunc("/{id}/add-to-cart", materialHandler.AddMaterialToCart).Methods("POST")
	materials.HandleFunc("/{id}/image", materialHandler.UploadMaterialImage).Methods("POST")

	// Домен расчёта
	calculations := apiV1.PathPrefix("/calculations").Subrouter()
	calculations.HandleFunc("/cart-info", calculationHandler.GetCartInfo).Methods("GET")
	calculations.HandleFunc("", calculationHandler.GetCalculations).Methods("GET")
	calculations.HandleFunc("/{id}", calculationHandler.GetCalculation).Methods("GET")
	calculations.HandleFunc("/{id}/form", calculationHandler.FormCalculation).Methods("PUT")
	calculations.HandleFunc("/{id}/status", calculationHandler.CompleteCalculation).Methods("PUT")
	calculations.HandleFunc("/{id}/materials", calculationHandler.GetCalculationMaterials).Methods("GET")
	calculations.HandleFunc("/{id}", calculationHandler.DeleteCalculation).Methods("DELETE")

	// Домен м-м (расчёт-материал)
	materialCalculations := apiV1.PathPrefix("/calculations/{calculationId}/materials").Subrouter()
	materialCalculations.HandleFunc("/{materialId}", materialCalculationHandler.DeleteMaterialCalculation).Methods("DELETE")
	materialCalculations.HandleFunc("/{materialId}", materialCalculationHandler.UpdateMaterialCalculation).Methods("PUT")

	// Домен пользователь
	users := apiV1.PathPrefix("/users").Subrouter()
	users.HandleFunc("/register", userHandler.Register).Methods("POST")
	users.HandleFunc("/login", userHandler.Login).Methods("POST")
	users.HandleFunc("/profile", userHandler.GetProfile).Methods("GET")
	users.HandleFunc("/profile", userHandler.UpdateProfile).Methods("PUT")
	users.HandleFunc("/logout", userHandler.Logout).Methods("POST")

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
