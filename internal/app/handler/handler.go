package handler

import (
	"encoding/json"
	"html/template"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"RIP/internal/app"
)

type Server struct {
	Store      *app.Store
	AssetsBase string
	Templates  *template.Template
}

func MustParseTemplates() *template.Template {
	var base string
	possiblePaths := []string{
		"templates",
		"RIP/templates",
		filepath.Join("..", "templates"),
	}
	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			base = p
			break
		}
	}
	if base == "" {
		panic("templates directory not found")
	}
	return template.Must(template.ParseFiles(
		filepath.Join(base, "catalog.html"),
		filepath.Join(base, "calc.html"),
		filepath.Join(base, "detail.html"),
	))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func (s *Server) HandleCatalog(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	thickness := strings.TrimSpace(r.URL.Query().Get("thickness"))
	reqID := r.URL.Query().Get("requestId")
	if reqID == "" {
		reqID = "1"
	}
	services := s.filterServices(q, thickness)
	count := s.cartCount(reqID)
	data := struct {
		Title      string
		Query      string
		Thickness  string
		RequestID  string
		AssetsBase string
		Services   []app.Service
		CartCount  int
	}{
		Title:      "Каталог материалов",
		Query:      q,
		Thickness:  thickness,
		RequestID:  reqID,
		AssetsBase: s.AssetsBase,
		Services:   services,
		CartCount:  count,
	}
	_ = s.Templates.ExecuteTemplate(w, "catalog.html", data)
}

func (s *Server) HandleCalc(w http.ResponseWriter, r *http.Request) {
	reqID := r.FormValue("requestId")
	if reqID == "" {
		reqID = r.URL.Query().Get("requestId")
	}
	if reqID == "" {
		reqID = "1"
	}
	items := append([]app.CartItem(nil), s.Store.Carts[reqID]...)

	var cartServices []app.Service
	for _, it := range items {
		if sv, ok := s.findService(it.ServiceID); ok {
			cartServices = append(cartServices, sv)
		}
	}
	var results []app.Result
	if r.Method == http.MethodPost {
		mass, _ := strconv.ParseFloat(r.FormValue("mass"), 64)
		freq, _ := strconv.ParseFloat(r.FormValue("frequency"), 64)
		results = s.calculateResults(items, mass, freq)
	}
	data := struct {
		Title        string
		RequestID    string
		AssetsBase   string
		CartItems    []app.CartItem
		CartServices []app.Service
		Results      []app.Result
		CartCount    int
	}{
		Title:        "Расчёт",
		RequestID:    reqID,
		AssetsBase:   s.AssetsBase,
		CartItems:    items,
		CartServices: cartServices,
		Results:      results,
		CartCount:    s.cartCount(reqID),
	}
	_ = s.Templates.ExecuteTemplate(w, "calc.html", data)
}

func (s *Server) HandleDetail(w http.ResponseWriter, r *http.Request) {
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
		Service    app.Service
		AssetsBase string
		RequestID  string
		CartCount  int
	}{
		Title:      svc.Name,
		Service:    svc,
		AssetsBase: s.AssetsBase,
		RequestID:  reqID,
		CartCount:  s.cartCount(reqID),
	}
	_ = s.Templates.ExecuteTemplate(w, "detail.html", data)
}

// API
func (s *Server) ApiServices(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	thickness := strings.TrimSpace(r.URL.Query().Get("thickness"))
	var list []app.Service
	for _, sv := range s.Store.Services {
		nameMatch := q == "" || strings.Contains(strings.ToLower(sv.Name), q)
		thicknessMatch := true
		if thickness != "" {
			thicknessMatch = false
			for _, prop := range sv.Props {
				if strings.Contains(strings.ToLower(prop), "толщина") && strings.Contains(strings.ToLower(prop), strings.ToLower(thickness)) {
					thicknessMatch = true
					break
				}
			}
		}
		if nameMatch && thicknessMatch {
			list = append(list, sv)
		}
	}
	writeJSON(w, map[string]any{"items": list, "count": len(list)})
}

func (s *Server) ApiAddToCart(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("requestId")
	serviceID, _ := strconv.Atoi(r.URL.Query().Get("serviceId"))
	items := s.Store.Carts[requestID]
	for i := range items {
		if items[i].ServiceID == serviceID {
			writeJSON(w, map[string]any{"success": false, "error": "already in cart", "items": items})
			return
		}
	}
	items = append(items, app.CartItem{ServiceID: serviceID, Quantity: 1})
	s.Store.Carts[requestID] = items
	writeJSON(w, map[string]any{"success": true, "items": items})
}

func (s *Server) ApiCart(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("requestId")
	items := append([]app.CartItem(nil), s.Store.Carts[requestID]...)
	writeJSON(w, map[string]any{"requestId": requestID, "items": items})
}

func (s *Server) ApiClearCart(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("requestId")
	s.Store.Carts[requestID] = []app.CartItem{}
	writeJSON(w, map[string]any{"success": true, "items": []app.CartItem{}})
}

func (s *Server) ApiCalc(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("requestId")
	mass, _ := strconv.ParseFloat(r.URL.Query().Get("mass"), 64)
	freq, _ := strconv.ParseFloat(r.URL.Query().Get("frequency"), 64)
	items := append([]app.CartItem(nil), s.Store.Carts[requestID]...)
	results := s.calculateResults(items, mass, freq)
	writeJSON(w, map[string]any{"requestId": requestID, "results": results})
}

// SSR actions (no JS)
func (s *Server) HandleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	requestID := r.FormValue("requestId")
	if requestID == "" {
		requestID = "1"
	}
	serviceID, _ := strconv.Atoi(r.FormValue("serviceId"))
	items := s.Store.Carts[requestID]
	exists := false
	for i := range items {
		if items[i].ServiceID == serviceID {
			exists = true
			break
		}
	}
	if !exists {
		items = append(items, app.CartItem{ServiceID: serviceID, Quantity: 1})
		s.Store.Carts[requestID] = items
	}
	ref := r.Referer()
	if ref == "" {
		ref = "/?requestId=" + requestID
	}
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

func (s *Server) HandleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	requestID := r.FormValue("requestId")
	if requestID == "" {
		requestID = "1"
	}
	s.Store.Carts[requestID] = []app.CartItem{}
	http.Redirect(w, r, "/calc?requestId="+requestID, http.StatusSeeOther)
}

// helpers
func (s *Server) findService(id int) (app.Service, bool) {
	for _, sv := range s.Store.Services {
		if sv.ID == id {
			return sv, true
		}
	}
	return app.Service{}, false
}

func (s *Server) filterServices(q, thickness string) []app.Service {
	q = strings.ToLower(strings.TrimSpace(q))
	thickness = strings.TrimSpace(thickness)
	var list []app.Service
	for _, sv := range s.Store.Services {
		nameMatch := q == "" || strings.Contains(strings.ToLower(sv.Name), q)
		thicknessMatch := true
		if thickness != "" {
			thicknessMatch = false
			for _, prop := range sv.Props {
				if strings.Contains(strings.ToLower(prop), "толщина") && strings.Contains(strings.ToLower(prop), strings.ToLower(thickness)) {
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
	sum := 0
	for _, it := range s.Store.Carts[requestID] {
		sum += it.Quantity
	}
	return sum
}

func (s *Server) calculateResults(items []app.CartItem, mass, freq float64) []app.Result {
	var results []app.Result
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
		results = append(results, app.Result{ServiceID: it.ServiceID, ServiceName: serviceName, NaturalHz: round(fn, 2), Isolation: round(iso, 1)})
	}
	return results
}

func round(x float64, p int) float64 {
	pow := math.Pow(10, float64(p))
	return math.Round(x*pow) / pow
}
