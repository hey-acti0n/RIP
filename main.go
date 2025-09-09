package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Domain models
type Service struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ImageURL    string   `json:"imageUrl"`
	Props       []string `json:"props"`
}

type CartItem struct {
	ServiceID int `json:"serviceId"`
	Quantity  int `json:"quantity"`
}

// In-memory storage (later replace with PostgreSQL)
type Store struct {
	services []Service
	// requestId -> items
	carts map[string][]CartItem
	mu    sync.RWMutex
}

func newStore() *Store {
	// MinIO is exposed at localhost:9000 by docker-compose.
	// For demo purposes, we assume bucket "images" and a few objects present.
	minioHost := getenv("MINIO_PUBLIC_ENDPOINT", "http://localhost:9000")
	mk := func(name, obj string) Service {
		return Service{
			ID:          len(name) + len(obj),
			Name:        name,
			Description: "Виброизоляционный материал для промышленного оборудования",
			ImageURL:    fmt.Sprintf("%s/%s/%s", minioHost, "images", obj),
			Props:       []string{"Плотность: 120 кг/м³", "Толщина: 10 мм", "Материал: EPDM"},
		}
	}
	return &Store{
		services: []Service{
			mk("Ultra Rezina", "ultra-rezina.jpg"),
			mk("Vibro Mat", "vibro-mat.jpg"),
			mk("Silent Pro", "silent-pro.jpg"),
			mk("Damp X", "damp-x.jpg"),
			mk("Iso Flex", "iso-flex.jpg"),
			mk("Soft Shield", "soft-shield.jpg"),
		},
		carts: map[string][]CartItem{},
	}
}

// Utilities
func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Templates
var tmpl *template.Template

func mustParseTemplates() *template.Template {
	// Пробуем разные пути для поиска templates
	var base string
	possiblePaths := []string{
		"templates",                      // текущая директория
		"RIP/templates",                  // если запускаем из Documents
		filepath.Join("..", "templates"), // если запускаем из подпапки
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			base = path
			break
		}
	}

	if base == "" {
		log.Fatal("Не удалось найти папку templates. Проверьте, что вы запускаете программу из правильной директории.")
	}

	t := template.Must(template.ParseFiles(
		filepath.Join(base, "catalog.html"),
		filepath.Join(base, "calc.html"),
		filepath.Join(base, "detail.html"),
	))
	return t
}

// Handlers (Controllers)
type Server struct {
	store      *Store
	assetsBase string
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	// q: search filter, requestId: to track cart
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	reqID := r.URL.Query().Get("requestId")
	if reqID == "" {
		reqID = "1" // demo default
	}
	data := struct {
		Title      string
		Query      string
		RequestID  string
		AssetsBase string
	}{
		Title:      "Каталог материалов",
		Query:      q,
		RequestID:  reqID,
		AssetsBase: s.assetsBase,
	}
	if err := tmpl.ExecuteTemplate(w, "catalog.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleCalc(w http.ResponseWriter, r *http.Request) {
	reqID := r.URL.Query().Get("requestId")
	if reqID == "" {
		reqID = "1"
	}
	data := struct {
		Title      string
		RequestID  string
		AssetsBase string
	}{Title: "Расчёт", RequestID: reqID, AssetsBase: s.assetsBase}
	if err := tmpl.ExecuteTemplate(w, "calc.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleDetail(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/detail/")
	id, _ := strconv.Atoi(idStr)
	svc, ok := s.findService(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	data := struct {
		Title      string
		Service    Service
		AssetsBase string
	}{
		Title:      svc.Name,
		Service:    svc,
		AssetsBase: s.assetsBase,
	}
	if err := tmpl.ExecuteTemplate(w, "detail.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// API endpoints to make GET requests visible in Network
func (s *Server) apiServices(w http.ResponseWriter, r *http.Request) {
	// GET /api/services?q=...&requestId=...
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	requestID := r.URL.Query().Get("requestId")
	_ = requestID // carried in response for demo
	var list []Service
	for _, sv := range s.store.services {
		if q == "" || strings.Contains(strings.ToLower(sv.Name), q) {
			list = append(list, sv)
		}
	}
	type resp struct {
		RequestID string    `json:"requestId"`
		Count     int       `json:"count"`
		Items     []Service `json:"items"`
	}
	writeJSON(w, resp{RequestID: requestID, Count: len(list), Items: list})
}

func (s *Server) apiAddToCart(w http.ResponseWriter, r *http.Request) {
	// GET /api/add?requestId=...&serviceId=...
	requestID := r.URL.Query().Get("requestId")
	serviceID, _ := strconv.Atoi(r.URL.Query().Get("serviceId"))
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	items := s.store.carts[requestID]
	for i := range items {
		if items[i].ServiceID == serviceID {
			// Товар уже в корзине
			writeJSON(w, map[string]any{
				"requestId": requestID,
				"items":     items,
				"error":     "Товар уже добавлен в корзину",
				"success":   false,
			})
			return
		}
	}
	items = append(items, CartItem{ServiceID: serviceID, Quantity: 1})
	s.store.carts[requestID] = items
	writeJSON(w, map[string]any{
		"requestId": requestID,
		"items":     items,
		"success":   true,
		"message":   "Товар добавлен в корзину",
	})
}

func (s *Server) apiCart(w http.ResponseWriter, r *http.Request) {
	// GET /api/cart?requestId=...
	requestID := r.URL.Query().Get("requestId")
	s.store.mu.RLock()
	items := append([]CartItem(nil), s.store.carts[requestID]...)
	s.store.mu.RUnlock()
	writeJSON(w, map[string]any{"requestId": requestID, "items": items})
}

func (s *Server) apiClearCart(w http.ResponseWriter, r *http.Request) {
	// GET /api/clear?requestId=...
	requestID := r.URL.Query().Get("requestId")
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	s.store.carts[requestID] = []CartItem{}
	writeJSON(w, map[string]any{
		"requestId": requestID,
		"items":     []CartItem{},
		"success":   true,
		"message":   "Корзина очищена",
	})
}

func (s *Server) apiCalc(w http.ResponseWriter, r *http.Request) {
	// GET /api/calc?requestId=...&mass=..&frequency=..
	requestID := r.URL.Query().Get("requestId")
	mass, _ := strconv.ParseFloat(r.URL.Query().Get("mass"), 64)
	freq, _ := strconv.ParseFloat(r.URL.Query().Get("frequency"), 64)

	s.store.mu.RLock()
	items := append([]CartItem(nil), s.store.carts[requestID]...)
	s.store.mu.RUnlock()

	// Simplified vibration isolation calculation demo per item
	type Result struct {
		ServiceID   int     `json:"serviceId"`
		ServiceName string  `json:"serviceName"`
		NaturalHz   float64 `json:"naturalHz"`
		Isolation   float64 `json:"isolationPercent"`
	}
	var results []Result
	for _, it := range items {
		// Находим название товара по ID
		serviceName := "Неизвестный товар"
		if service, ok := s.findService(it.ServiceID); ok {
			serviceName = service.Name
		}

		// demo formula: fn = sqrt(k/m), assume k depends on quantity
		stiffness := 1000.0 * float64(it.Quantity)
		if mass <= 0 {
			mass = 1
		}
		fn := math.Sqrt(stiffness/mass) / (2 * math.Pi)
		iso := 100.0 * (1 - (fn / (freq + fn)))
		results = append(results, Result{
			ServiceID:   it.ServiceID,
			ServiceName: serviceName,
			NaturalHz:   round(fn, 2),
			Isolation:   round(iso, 1),
		})
	}
	writeJSON(w, map[string]any{"requestId": requestID, "results": results})
}

// Helpers
func (s *Server) findService(id int) (Service, bool) {
	for _, sv := range s.store.services {
		if sv.ID == id {
			return sv, true
		}
	}
	return Service{}, false
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func round(x float64, p int) float64 {
	pow := math.Pow(10, float64(p))
	return math.Round(x*pow) / pow
}

func main() {
	tmpl = mustParseTemplates()
	minioHost := getenv("MINIO_PUBLIC_ENDPOINT", "http://localhost:9000")
	assetsBase := fmt.Sprintf("%s/%s", minioHost, "images")
	srv := &Server{store: newStore(), assetsBase: assetsBase}

	// Находим путь к статическим файлам
	var staticDir string
	possibleStaticPaths := []string{
		"static",                      // текущая директория
		"RIP/static",                  // если запускаем из Documents
		filepath.Join("..", "static"), // если запускаем из подпапки
	}

	for _, path := range possibleStaticPaths {
		if _, err := os.Stat(path); err == nil {
			staticDir = path
			break
		}
	}

	if staticDir == "" {
		staticDir = "static" // fallback
	}

	fs := http.FileServer(http.Dir(staticDir))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Routing: 3 pages
	http.HandleFunc("/", srv.handleCatalog)
	http.HandleFunc("/calc", srv.handleCalc)
	http.HandleFunc("/detail/", srv.handleDetail)

	// API: 4 example GET requests for demo
	http.HandleFunc("/api/services", srv.apiServices)
	http.HandleFunc("/api/add", srv.apiAddToCart)
	http.HandleFunc("/api/cart", srv.apiCart)
	http.HandleFunc("/api/clear", srv.apiClearCart)
	http.HandleFunc("/api/calc", srv.apiCalc)

	addr := getenv("ADDR", ":8080")
	log.Printf("UltraRezina server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))

}
