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
	serviceHandler := NewServiceHandler(r.service)
	requestHandler := NewRequestHandler(r.service)
	requestServiceHandler := NewRequestServiceHandler(r.service)
	userHandler := NewUserHandler(r.service)

	// API v1
	apiV1 := router.PathPrefix("/api/v1").Subrouter()

	// Домен услуги
	services := apiV1.PathPrefix("/services").Subrouter()
	services.HandleFunc("", serviceHandler.GetServices).Methods("GET")
	services.HandleFunc("/{id}", serviceHandler.GetService).Methods("GET")
	services.HandleFunc("", serviceHandler.CreateService).Methods("POST")
	services.HandleFunc("/{id}", serviceHandler.UpdateService).Methods("PUT")
	services.HandleFunc("/{id}", serviceHandler.DeleteService).Methods("DELETE")
	services.HandleFunc("/{id}/add-to-cart", serviceHandler.AddServiceToCart).Methods("POST")
	services.HandleFunc("/{id}/image", serviceHandler.UploadServiceImage).Methods("POST")

	// Домен заявки
	requests := apiV1.PathPrefix("/requests").Subrouter()
	requests.HandleFunc("/cart-info", requestHandler.GetCartInfo).Methods("GET")
	requests.HandleFunc("", requestHandler.GetRequests).Methods("GET")
	requests.HandleFunc("/{id}", requestHandler.GetRequest).Methods("GET")
	requests.HandleFunc("/{id}", requestHandler.UpdateRequest).Methods("PUT")
	requests.HandleFunc("/{id}/form", requestHandler.FormRequest).Methods("PUT")
	requests.HandleFunc("/{id}/status", requestHandler.CompleteRequest).Methods("PUT")
	requests.HandleFunc("/{id}/services", requestHandler.GetRequestServices).Methods("GET")
	requests.HandleFunc("/{id}", requestHandler.DeleteRequest).Methods("DELETE")

	// Домен м-м (заявка-услуга)
	requestServices := apiV1.PathPrefix("/requests/{requestId}/services").Subrouter()
	requestServices.HandleFunc("/{serviceId}", requestServiceHandler.DeleteRequestService).Methods("DELETE")
	requestServices.HandleFunc("/{serviceId}", requestServiceHandler.UpdateRequestService).Methods("PUT")

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
