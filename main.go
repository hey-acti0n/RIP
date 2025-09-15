package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gorm.io/gorm"
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

// Calculation result (used for API and SSR)
type Result struct {
	ServiceID   int     `json:"serviceId"`
	ServiceName string  `json:"serviceName"`
	NaturalHz   float64 `json:"naturalHz"`
	Isolation   float64 `json:"isolationPercent"`
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
	db         *gorm.DB
}

// buildProps builds a list of characteristics from DB values
func buildProps(dbSvc DBService) []string {
	var props []string
	if dbSvc.Density != nil {
		props = append(props, fmt.Sprintf("Плотность: %.2f", *dbSvc.Density))
	}
	if dbSvc.Thickness != nil {
		props = append(props, fmt.Sprintf("Толщина: %.2f мм", *dbSvc.Thickness))
	}
	if dbSvc.Material != nil && *dbSvc.Material != "" {
		props = append(props, fmt.Sprintf("Материал: %s", *dbSvc.Material))
	}
	return props
}

func (s *Server) handleCatalog(w http.ResponseWriter, r *http.Request) {
	// SSR catalog with filters and cart badge
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	thickness := strings.TrimSpace(r.URL.Query().Get("thickness"))
	reqID := r.URL.Query().Get("requestId")
	if reqID == "" {
		reqID = "1" // demo default
	}

	services := s.filterServices(q, thickness)
	count := s.cartCount(reqID)

	data := struct {
		Title      string
		Query      string
		Thickness  string
		RequestID  string
		AssetsBase string
		Services   []Service
		CartCount  int
	}{
		Title:      "Каталог материалов",
		Query:      q,
		Thickness:  thickness,
		RequestID:  reqID,
		AssetsBase: s.assetsBase,
		Services:   services,
		CartCount:  count,
	}
	if err := tmpl.ExecuteTemplate(w, "catalog.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleCalc(w http.ResponseWriter, r *http.Request) {
	// GET renders page. POST performs calculation and shows results.
	reqID := r.FormValue("requestId")
	if reqID == "" {
		reqID = r.URL.Query().Get("requestId")
	}
	if reqID == "" {
		reqID = "1"
	}

	var items []CartItem
	var cartServices []Service
	if s.db != nil {
		var rs []RequestService
		_ = s.db.Preload("Service").Where("request_id = ?", reqID).Find(&rs).Error
		for _, it := range rs {
			items = append(items, CartItem{ServiceID: it.ServiceID, Quantity: it.Quantity})
			props := buildProps(it.Service)
			cartServices = append(cartServices, Service{ID: it.Service.ID, Name: it.Service.Name, Description: it.Service.Description, ImageURL: it.Service.ImageURL, Props: props})
		}
	} else {
		s.store.mu.RLock()
		items = append([]CartItem(nil), s.store.carts[reqID]...)
		s.store.mu.RUnlock()
		for _, it := range items {
			if sv, ok := s.findService(it.ServiceID); ok {
				cartServices = append(cartServices, sv)
			}
		}
	}

	var results []Result
	if r.Method == http.MethodPost {
		mass, _ := strconv.ParseFloat(r.FormValue("mass"), 64)
		freq, _ := strconv.ParseFloat(r.FormValue("frequency"), 64)
		results = s.calculateResults(items, mass, freq)
		// смена статуса заявки на completed в БД
		if s.db != nil {
			_ = s.db.Exec("UPDATE requests SET status = 'completed' WHERE id = ?", reqID).Error
		}
	}

	data := struct {
		Title        string
		RequestID    string
		AssetsBase   string
		CartItems    []CartItem
		CartServices []Service
		Results      []Result
		CartCount    int
	}{
		Title:        "Расчёт",
		RequestID:    reqID,
		AssetsBase:   s.assetsBase,
		CartItems:    items,
		CartServices: cartServices,
		Results:      results,
		CartCount:    s.cartCount(reqID),
	}
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
	reqID := r.URL.Query().Get("requestId")
	if reqID == "" {
		reqID = "1"
	}
	data := struct {
		Title      string
		Service    Service
		AssetsBase string
		RequestID  string
		CartCount  int
	}{
		Title:      svc.Name,
		Service:    svc,
		AssetsBase: s.assetsBase,
		RequestID:  reqID,
		CartCount:  s.cartCount(reqID),
	}
	if err := tmpl.ExecuteTemplate(w, "detail.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// API endpoints to make GET requests visible in Network
func (s *Server) apiServices(w http.ResponseWriter, r *http.Request) {
	// GET /api/services?q=...&thickness=...&requestId=...
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	thickness := strings.TrimSpace(r.URL.Query().Get("thickness"))
	requestID := r.URL.Query().Get("requestId")
	_ = requestID // carried in response for demo
	var list []Service
	for _, sv := range s.store.services {
		// Фильтр по названию
		nameMatch := q == "" || strings.Contains(strings.ToLower(sv.Name), q)

		// Фильтр по толщине
		thicknessMatch := true
		if thickness != "" {
			thicknessMatch = false
			for _, prop := range sv.Props {
				if strings.Contains(strings.ToLower(prop), "толщина") &&
					strings.Contains(strings.ToLower(prop), strings.ToLower(thickness)) {
					thicknessMatch = true
					break
				}
			}
		}

		if nameMatch && thicknessMatch {
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

	results := s.calculateResults(items, mass, freq)
	writeJSON(w, map[string]any{"requestId": requestID, "results": results})
}

// Helpers
func (s *Server) findService(id int) (Service, bool) {
	if s.db != nil {
		var d DBService
		if err := s.db.First(&d, id).Error; err != nil {
			return Service{}, false
		}
		return Service{ID: d.ID, Name: d.Name, Description: d.Description, ImageURL: d.ImageURL, Props: buildProps(d)}, true
	}
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

// setURLParam заменяет или добавляет query-параметр в URL
func setURLParam(rawURL, key, val string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	q.Set(key, val)
	u.RawQuery = q.Encode()
	return u.String()
}

// Helpers for SSR
func (s *Server) filterServices(q, thickness string) []Service {
	q = strings.ToLower(strings.TrimSpace(q))
	thickness = strings.TrimSpace(thickness)
	// ORM-backed filter when DB is available
	if s.db != nil {
		var listDB []DBService
		tx := s.db.Model(&DBService{}).Where("is_active = ?", true)
		if q != "" {
			tx = tx.Where("LOWER(name) LIKE ?", "%"+q+"%")
		}
		if thickness != "" {
			tx = tx.Where("CAST(thickness AS TEXT) LIKE ?", "%"+thickness+"%")
		}
		_ = tx.Find(&listDB).Error
		var out []Service
		for _, d := range listDB {
			out = append(out, Service{ID: d.ID, Name: d.Name, Description: d.Description, ImageURL: d.ImageURL, Props: buildProps(d)})
		}
		return out
	}
	var list []Service
	for _, sv := range s.store.services {
		nameMatch := q == "" || strings.Contains(strings.ToLower(sv.Name), q)
		thicknessMatch := true
		if thickness != "" {
			thicknessMatch = false
			for _, prop := range sv.Props {
				if strings.Contains(strings.ToLower(prop), "толщина") &&
					strings.Contains(strings.ToLower(prop), strings.ToLower(thickness)) {
					thicknessMatch = true
					break
				}
			}
		}
		if nameMatch && thicknessMatch {
			list = append(list, sv)
		}
	}
	return list
}

func (s *Server) cartCount(requestID string) int {
	if s.db != nil {
		var sum int64
		_ = s.db.Model(&RequestService{}).Where("request_id = ?", requestID).Select("COALESCE(SUM(quantity),0)").Scan(&sum).Error
		return int(sum)
	}
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	sum := 0
	for _, it := range s.store.carts[requestID] {
		sum += it.Quantity
	}
	return sum
}

func (s *Server) calculateResults(items []CartItem, mass, freq float64) []Result {
	var results []Result
	for _, it := range items {
		serviceName := "Неизвестный товар"
		if service, ok := s.findService(it.ServiceID); ok {
			serviceName = service.Name
		}
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
	return results
}

// Actions for SSR (forms instead of JS)
func (s *Server) handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	requestID := r.FormValue("requestId")
	serviceID, _ := strconv.Atoi(r.FormValue("serviceId"))
	log.Printf("/add POST: requestId=%q serviceId=%d", requestID, serviceID)
	if s.db != nil {
		// userId условно 1 (можно взять из сессии позже)
		userID := 1
		var req Request
		err := s.db.Where("id = ? AND status <> 'rejected'", requestID).First(&req).Error
		if err == gorm.ErrRecordNotFound || requestID == "" {
			// ищем черновик пользователя либо создаём
			if err := s.db.Where("creator_id = ? AND status = 'pending'", userID).First(&req).Error; err == gorm.ErrRecordNotFound {
				if err := s.db.Model(&Request{}).Create(map[string]any{"creator_id": userID, "status": "pending"}).Error; err != nil {
					log.Printf("create draft request error: %v", err)
					http.Error(w, "db error", http.StatusInternalServerError)
					return
				}
				_ = s.db.Where("creator_id = ?", userID).Order("id desc").First(&req).Error
			}
			requestID = strconv.Itoa(req.ID)
		}
		// upsert позиции
		if err := s.db.Exec("INSERT INTO request_services (request_id, service_id, quantity) VALUES (?, ?, 1) ON CONFLICT (request_id, service_id) DO UPDATE SET quantity = request_services.quantity + 1", requestID, serviceID).Error; err != nil {
			log.Printf("add to request_services error: %v", err)
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		// гарантируем статус pending у заявки
		_ = s.db.Exec("UPDATE requests SET status = 'pending' WHERE id = ? AND status <> 'pending'", requestID).Error
	} else {
		if requestID == "" {
			requestID = "1"
		}
		s.store.mu.Lock()
		items := s.store.carts[requestID]
		exists := false
		for i := range items {
			if items[i].ServiceID == serviceID {
				exists = true
				break
			}
		}
		if !exists {
			items = append(items, CartItem{ServiceID: serviceID, Quantity: 1})
			s.store.carts[requestID] = items
		} else {
			s.store.carts[requestID] = items
		}
		s.store.mu.Unlock()
	}

	// Всегда остаёмся на каталоге после добавления
	ref := r.Referer()
	if ref == "" {
		ref = "/?requestId=" + requestID
	} else {
		ref = setURLParam(ref, "requestId", requestID)
	}
	log.Printf("/add redirect -> %s", ref)
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

func (s *Server) handleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	requestID := r.FormValue("requestId")
	if s.db != nil {
		// помечаем заявку как rejected и очищаем её позиции
		_ = s.db.Exec("UPDATE requests SET status = 'rejected' WHERE id = ?", requestID).Error
		_ = s.db.Exec("DELETE FROM request_services WHERE request_id = ?", requestID).Error
	} else {
		if requestID == "" {
			requestID = "1"
		}
		s.store.mu.Lock()
		s.store.carts[requestID] = []CartItem{}
		s.store.mu.Unlock()
	}
	http.Redirect(w, r, "/calc?requestId="+requestID, http.StatusSeeOther)
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

	// Routing: pages
	http.HandleFunc("/", srv.handleCatalog)
	http.HandleFunc("/calc", srv.handleCalc)
	http.HandleFunc("/detail/", srv.handleDetail)

	// SSR actions (no JS)
	http.HandleFunc("/add", srv.handleAdd)
	http.HandleFunc("/clear", srv.handleClear)

	// API: 4 example GET requests for demo
	http.HandleFunc("/api/services", srv.apiServices)
	http.HandleFunc("/api/add", srv.apiAddToCart)
	http.HandleFunc("/api/cart", srv.apiCart)
	http.HandleFunc("/api/clear", srv.apiClearCart)
	http.HandleFunc("/api/calc", srv.apiCalc)

	// Подключаем БД и монтируем ORM/SQL маршруты
	db := InitDB()
	srv.db = db
	MountORMRoutes(db)

	addr := getenv("ADDR", ":8080")
	log.Printf("UltraRezina server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))

}
