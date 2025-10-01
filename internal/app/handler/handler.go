package handler

import (
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"rip/internal/app/repository"

	"gorm.io/gorm"
)

type Material struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ImageURL    string   `json:"imageUrl"`
	Props       []string `json:"props"`
	Comment     string   `json:"comment"`
}

type CartItem struct {
	MaterialID int    `json:"materialId"`
	Quantity   int    `json:"quantity"`
	Comment    string `json:"comment"`
}

type Result struct {
	MaterialID   int     `json:"materialId"`
	MaterialName string  `json:"materialName"`
	NaturalHz    float64 `json:"naturalHz"`
	Isolation    float64 `json:"isolationPercent"`
}

type Store struct {
	materials []Material
	carts     map[string][]CartItem
	mu        sync.RWMutex
}

func newStore(assetsBase string) *Store {
	mk := func(name, obj string) Material {
		return Material{
			ID:          len(name) + len(obj),
			Name:        name,
			Description: "Виброизоляционный материал для промышленного оборудования",
			ImageURL:    fmt.Sprintf("%s/%s", assetsBase, obj),
			Props:       []string{"Плотность: 120 кг/м³", "Толщина: 10 мм", "Материал: EPDM"},
		}
	}
	return &Store{
		materials: []Material{
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

type Server struct {
	store      *Store
	assetsBase string
	db         *gorm.DB
	tmpl       *template.Template
}

func NewServer(assetsBase string, tmpl *template.Template) *Server {
	return &Server{store: newStore(assetsBase), assetsBase: assetsBase, tmpl: tmpl}
}

func (s *Server) AttachDB(db *gorm.DB) { s.db = db }

func buildProps(dbMaterial repository.DBMaterial) []string {
	var props []string
	if dbMaterial.Density != nil {
		props = append(props, fmt.Sprintf("Плотность: %.2f", *dbMaterial.Density))
	}
	if dbMaterial.Thickness != nil {
		props = append(props, fmt.Sprintf("Толщина: %.2f мм", *dbMaterial.Thickness))
	}
	if dbMaterial.Material != nil && *dbMaterial.Material != "" {
		props = append(props, fmt.Sprintf("Материал: %s", *dbMaterial.Material))
	}
	return props
}

func (s *Server) HandleCatalog(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	thickness := strings.TrimSpace(r.URL.Query().Get("thickness"))

	// Проверяем, есть ли ID в пути (например, /56)
	path := strings.TrimPrefix(r.URL.Path, "/")
	reqID := r.URL.Query().Get("requestId")
	if reqID == "" && path != "" {
		// Если ID в пути, извлекаем последнюю часть (например, из "1/57" получаем "57")
		pathParts := strings.Split(path, "/")
		if len(pathParts) > 0 {
			reqID = pathParts[len(pathParts)-1]
		}
	}
	if reqID == "" {
		reqID = "1"
	}
	materials := s.filterMaterials(q, thickness)
	count := s.cartCount(reqID)
	data := struct {
		Title      string
		Query      string
		Thickness  string
		RequestID  string
		AssetsBase string
		Materials  []Material
		CartCount  int
	}{
		Title:      "Каталог материалов",
		Query:      q,
		Thickness:  thickness,
		RequestID:  reqID,
		AssetsBase: s.assetsBase,
		Materials:  materials,
		CartCount:  count,
	}
	if err := s.tmpl.ExecuteTemplate(w, "catalog.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) HandleCalc(w http.ResponseWriter, r *http.Request) {
	reqID := r.FormValue("requestId")
	if reqID == "" {
		reqID = r.URL.Query().Get("requestId")
	}
	if reqID == "" {
		reqID = "1"
	}
	var items []CartItem
	var cartMaterials []Material
	var results []Result
	if s.db != nil {
		// Проверяем статус заявки - если rejected, то корзина пустая
		var calculation repository.Calculation
		if err := s.db.Where("id = ?", reqID).First(&calculation).Error; err == nil {
			if calculation.Status == "rejected" {
				// Заявка отклонена, корзина пустая
				items = []CartItem{}
				cartMaterials = []Material{}
			} else {
				// Заявка активна, загружаем товары
				var rs []repository.MaterialCalculation
				_ = s.db.Preload("Material").Where("calculation_id = ?", reqID).Find(&rs).Error
				for _, it := range rs {
					items = append(items, CartItem{MaterialID: it.MaterialID, Quantity: it.Quantity})
					props := buildProps(it.Material)
					imageURL := it.Material.ImageURL
					if !strings.HasPrefix(imageURL, "http") {
						if strings.HasPrefix(imageURL, "/images/") {
							imageURL = "http://localhost:9000" + imageURL
						} else {
							imageURL = s.assetsBase + imageURL
						}
					}
					service := Material{ID: it.Material.ID, Name: it.Material.Name, Description: it.Material.Description, ImageURL: imageURL, Props: props, Comment: it.Comment}
					cartMaterials = append(cartMaterials, service)

					// Если есть сохраненные результаты, добавляем их
					if it.ResultFreq != nil && it.ResultPercent != nil {
						results = append(results, Result{
							MaterialID:   it.MaterialID,
							MaterialName: it.Material.Name,
							NaturalHz:    *it.ResultFreq,
							Isolation:    *it.ResultPercent,
						})
					}
				}
			}
		}
	} else {
		s.store.mu.RLock()
		items = append([]CartItem(nil), s.store.carts[reqID]...)
		s.store.mu.RUnlock()
		for _, it := range items {
			if sv, ok := s.findMaterial(it.MaterialID); ok {
				sv.Comment = it.Comment
				cartMaterials = append(cartMaterials, sv)
			}
		}
	}
	if r.Method == http.MethodPost {
		// Обрабатываем комментарии для товаров (всегда при POST)
		if s.db != nil {
			for _, service := range cartMaterials {
				commentKey := fmt.Sprintf("comment_%d", service.ID)
				comment := r.FormValue(commentKey)
				// Обновляем комментарий в базе данных (даже если пустой)
				_ = s.db.Exec("UPDATE material_calculation SET comment = ? WHERE calculation_id = ? AND material_id = ?",
					comment, reqID, service.ID).Error
			}

			// Перезагружаем данные из базы с обновленными комментариями
			var rs []repository.MaterialCalculation
			_ = s.db.Preload("Material").Where("calculation_id = ?", reqID).Find(&rs).Error
			cartMaterials = []Material{} // Очищаем старые данные
			items = []CartItem{}         // Очищаем старые данные
			for _, it := range rs {
				items = append(items, CartItem{MaterialID: it.MaterialID, Quantity: it.Quantity})
				props := buildProps(it.Material)
				imageURL := it.Material.ImageURL
				if !strings.HasPrefix(imageURL, "http") {
					if strings.HasPrefix(imageURL, "/images/") {
						imageURL = "http://localhost:9000" + imageURL
					} else {
						imageURL = s.assetsBase + imageURL
					}
				}
				service := Material{ID: it.Material.ID, Name: it.Material.Name, Description: it.Material.Description, ImageURL: imageURL, Props: props, Comment: it.Comment}
				cartMaterials = append(cartMaterials, service)
			}
		} else {
			// Обрабатываем комментарии для in-memory store
			s.store.mu.Lock()
			for i := range s.store.carts[reqID] {
				commentKey := fmt.Sprintf("comment_%d", s.store.carts[reqID][i].MaterialID)
				comment := r.FormValue(commentKey)
				s.store.carts[reqID][i].Comment = comment
			}
			s.store.mu.Unlock()

			// Обновляем cartMaterials с новыми комментариями
			cartMaterials = []Material{}
			for _, it := range items {
				if sv, ok := s.findMaterial(it.MaterialID); ok {
					sv.Comment = it.Comment
					cartMaterials = append(cartMaterials, sv)
				}
			}
		}

		// Проверяем, есть ли данные для расчета (масса и частота)
		mass, _ := strconv.ParseFloat(r.FormValue("mass"), 64)
		freq, _ := strconv.ParseFloat(r.FormValue("frequency"), 64)

		if mass > 0 && freq > 0 {
			// Если есть данные для расчета, выполняем расчет
			results = s.calculateResults(items, mass, freq)
			if s.db != nil {
				_ = s.db.Exec("UPDATE calculations SET status = 'completed' WHERE id = ?", reqID).Error

				// Сохраняем результаты расчета в базу данных
				for _, result := range results {
					_ = s.db.Exec("UPDATE material_calculation SET result_freq = ?, result_percent = ? WHERE calculation_id = ? AND material_id = ?",
						result.NaturalHz, result.Isolation, reqID, result.MaterialID).Error
				}
			}
		}
	}
	data := struct {
		Title         string
		RequestID     string
		AssetsBase    string
		CartItems     []CartItem
		CartMaterials []Material
		Results       []Result
		CartCount     int
	}{
		Title:         "Расчёт",
		RequestID:     reqID,
		AssetsBase:    s.assetsBase,
		CartItems:     items,
		CartMaterials: cartMaterials,
		Results:       results,
		CartCount:     s.cartCount(reqID),
	}
	if err := s.tmpl.ExecuteTemplate(w, "calc.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) HandleDetail(w http.ResponseWriter, r *http.Request) {
	// Обрабатываем новый формат URL: /detail/{id}/{requestId}
	path := strings.TrimPrefix(r.URL.Path, "/detail/")
	parts := strings.Split(path, "/")

	var idStr, reqID string
	if len(parts) >= 2 {
		idStr = parts[0]
		reqID = parts[1]
	} else if len(parts) == 1 {
		idStr = parts[0]
		reqID = r.URL.Query().Get("requestId")
	}

	if reqID == "" {
		reqID = "1"
	}

	id, _ := strconv.Atoi(idStr)
	svc, ok := s.findMaterial(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	data := struct {
		Title      string
		Material   Material
		AssetsBase string
		RequestID  string
		CartCount  int
	}{
		Title:      svc.Name,
		Material:   svc,
		AssetsBase: s.assetsBase,
		RequestID:  reqID,
		CartCount:  s.cartCount(reqID),
	}
	if err := s.tmpl.ExecuteTemplate(w, "detail.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// API
func (s *Server) ApiMaterials(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	thickness := strings.TrimSpace(r.URL.Query().Get("thickness"))
	requestID := r.URL.Query().Get("requestId")
	_ = requestID
	var list []Material
	for _, sv := range s.store.materials {
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
	repository.WriteJSON(w, map[string]any{"requestId": requestID, "count": len(list), "items": list})
}

func (s *Server) ApiMaterialByID(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID из URL
	path := strings.TrimPrefix(r.URL.Path, "/api/services/")
	id, err := strconv.Atoi(path)
	if err != nil {
		http.Error(w, "Invalid service ID", http.StatusBadRequest)
		return
	}

	// Ищем услугу по ID
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	for _, sv := range s.store.materials {
		if sv.ID == id {
			repository.WriteJSON(w, sv)
			return
		}
	}

	http.Error(w, "Material not found", http.StatusNotFound)
}

func (s *Server) ApiAddToCart(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("requestId")
	serviceID, _ := strconv.Atoi(r.URL.Query().Get("serviceId"))
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	items := s.store.carts[requestID]
	for i := range items {
		if items[i].MaterialID == serviceID {
			repository.WriteJSON(w, map[string]any{"requestId": requestID, "items": items, "error": "Товар уже добавлен в корзину", "success": false})
			return
		}
	}
	items = append(items, CartItem{MaterialID: serviceID, Quantity: 1})
	s.store.carts[requestID] = items
	repository.WriteJSON(w, map[string]any{"requestId": requestID, "items": items, "success": true, "message": "Товар добавлен в корзину"})
}

func (s *Server) ApiCart(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("requestId")
	s.store.mu.RLock()
	items := append([]CartItem(nil), s.store.carts[requestID]...)
	s.store.mu.RUnlock()
	repository.WriteJSON(w, map[string]any{"requestId": requestID, "items": items})
}

func (s *Server) ApiClearCart(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("requestId")
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	s.store.carts[requestID] = []CartItem{}
	repository.WriteJSON(w, map[string]any{"requestId": requestID, "items": []CartItem{}, "success": true, "message": "Корзина очищена"})
}

func (s *Server) ApiCalc(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("requestId")
	mass, _ := strconv.ParseFloat(r.URL.Query().Get("mass"), 64)
	freq, _ := strconv.ParseFloat(r.URL.Query().Get("frequency"), 64)
	s.store.mu.RLock()
	items := append([]CartItem(nil), s.store.carts[requestID]...)
	s.store.mu.RUnlock()
	results := s.calculateResults(items, mass, freq)
	repository.WriteJSON(w, map[string]any{"requestId": requestID, "results": results})
}

// helpers
func (s *Server) findMaterial(id int) (Material, bool) {
	if s.db != nil {
		var d repository.DBMaterial
		if err := s.db.First(&d, id).Error; err != nil {
			return Material{}, false
		}
		imageURL := d.ImageURL
		if !strings.HasPrefix(imageURL, "http") {
			if strings.HasPrefix(imageURL, "/images/") {
				imageURL = "http://localhost:9000" + imageURL
			} else {
				imageURL = s.assetsBase + imageURL
			}
		}
		return Material{ID: d.ID, Name: d.Name, Description: d.Description, ImageURL: imageURL, Props: buildProps(d)}, true
	}
	for _, sv := range s.store.materials {
		if sv.ID == id {
			return sv, true
		}
	}
	return Material{}, false
}

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

func (s *Server) filterMaterials(q, thickness string) []Material {
	q = strings.ToLower(strings.TrimSpace(q))
	thickness = strings.TrimSpace(thickness)
	if s.db != nil {
		var listDB []repository.DBMaterial
		tx := s.db.Model(&repository.DBMaterial{}).Where("is_active = ?", true)
		if q != "" {
			tx = tx.Where("LOWER(name) LIKE ?", "%"+q+"%")
		}
		if thickness != "" {
			tx = tx.Where("CAST(thickness AS TEXT) LIKE ?", "%"+thickness+"%")
		}
		_ = tx.Find(&listDB).Error
		var out []Material
		for _, d := range listDB {
			imageURL := d.ImageURL
			if !strings.HasPrefix(imageURL, "http") {
				if strings.HasPrefix(imageURL, "/images/") {
					imageURL = "http://localhost:9000" + imageURL
				} else {
					imageURL = s.assetsBase + imageURL
				}
			}
			out = append(out, Material{ID: d.ID, Name: d.Name, Description: d.Description, ImageURL: imageURL, Props: buildProps(d)})
		}
		return out
	}
	var list []Material
	for _, sv := range s.store.materials {
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
	if s.db != nil {
		// Проверяем статус заявки - если rejected, то корзина пустая
		var calculation repository.Calculation
		if err := s.db.Where("id = ?", requestID).First(&calculation).Error; err == nil {
			if calculation.Status == "rejected" {
				return 0
			}
		}
		var sum int64
		_ = s.db.Model(&repository.MaterialCalculation{}).Where("calculation_id = ?", requestID).Select("COALESCE(SUM(quantity),0)").Scan(&sum).Error
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
		if service, ok := s.findMaterial(it.MaterialID); ok {
			serviceName = service.Name
		}
		stiffness := 1000.0 * float64(it.Quantity)
		if mass <= 0 {
			mass = 1
		}
		fn := math.Sqrt(stiffness/mass) / (2 * math.Pi)
		iso := 100.0 * (1 - (fn / (freq + fn)))
		results = append(results, Result{MaterialID: it.MaterialID, MaterialName: serviceName, NaturalHz: round(fn, 2), Isolation: round(iso, 1)})
	}
	return results
}

func (s *Server) HandleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	requestID := r.FormValue("requestId")
	serviceID, _ := strconv.Atoi(r.FormValue("serviceId"))

	// Если requestID пустой, попробуем извлечь из referer URL
	if requestID == "" {
		referer := r.Referer()
		if referer != "" {
			// Извлекаем ID из referer URL (например, из http://localhost:8080/57)
			if strings.Contains(referer, "://") {
				parts := strings.Split(referer, "://")
				if len(parts) > 1 {
					pathParts := strings.Split(parts[1], "/")
					if len(pathParts) > 1 && pathParts[1] != "" {
						// Извлекаем последнюю часть пути (например, из "1/57" получаем "57")
						lastPart := pathParts[len(pathParts)-1]
						if lastPart != "" {
							requestID = lastPart
						}
					}
				}
			}
		}
	}

	if requestID == "" {
		requestID = "1"
	}

	log.Printf("/add POST: requestId=%q serviceId=%d", requestID, serviceID)
	if s.db != nil {
		userID := 1
		var req repository.Calculation
		err := s.db.Where("id = ? AND status <> 'rejected'", requestID).First(&req).Error
		if err == gorm.ErrRecordNotFound {
			if err := s.db.Where("creator_id = ? AND status = 'pending'", userID).First(&req).Error; err == gorm.ErrRecordNotFound {
				if err := s.db.Model(&repository.Calculation{}).Create(map[string]any{"creator_id": userID, "status": "pending"}).Error; err != nil {
					log.Printf("create draft request error: %v", err)
					http.Error(w, "db error", http.StatusInternalServerError)
					return
				}
				_ = s.db.Where("creator_id = ?", userID).Order("id desc").First(&req).Error
			}
			requestID = strconv.Itoa(req.ID)
		} else if err == nil {
			// Заявка найдена, используем её ID
			requestID = strconv.Itoa(req.ID)
		}
		if err := s.db.Exec("INSERT INTO material_calculation (calculation_id, material_id, quantity) VALUES (?, ?, 1) ON CONFLICT (calculation_id, material_id) DO UPDATE SET quantity = material_calculation.quantity + 1", requestID, serviceID).Error; err != nil {
			log.Printf("add to material_calculation error: %v", err)
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		_ = s.db.Exec("UPDATE calculations SET status = 'pending' WHERE id = ? AND status <> 'pending'", requestID).Error
	} else {
		s.store.mu.Lock()
		items := s.store.carts[requestID]
		exists := false
		for i := range items {
			if items[i].MaterialID == serviceID {
				exists = true
				break
			}
		}
		if !exists {
			items = append(items, CartItem{MaterialID: serviceID, Quantity: 1})
			s.store.carts[requestID] = items
		} else {
			s.store.carts[requestID] = items
		}
		s.store.mu.Unlock()
	}
	ref := r.Referer()
	if ref == "" {
		ref = "/" + requestID
	} else {
		// Обновляем URL в referer для нового формата
		if strings.Contains(ref, "?requestId=") {
			ref = strings.Split(ref, "?")[0] + "/" + requestID
		} else {
			// Заменяем ID в URL на актуальный requestID
			if strings.Contains(ref, "://") {
				parts := strings.Split(ref, "://")
				if len(parts) > 1 {
					pathParts := strings.Split(parts[1], "/")
					if len(pathParts) > 1 {
						pathParts[1] = requestID
						ref = parts[0] + "://" + strings.Join(pathParts, "/")
					}
				}
			}
		}
	}
	log.Printf("/add redirect -> %s", ref)
	http.Redirect(w, r, ref, http.StatusSeeOther)
}

func (s *Server) HandleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	requestID := r.FormValue("requestId")
	if s.db != nil {
		_ = s.db.Exec("UPDATE calculations SET status = 'rejected' WHERE id = ?", requestID).Error
		//_ = s.db.Exec("DELETE FROM material_calculation WHERE calculation_id = ?", requestID).Error
	} else {
		if requestID == "" {
			requestID = "1"
		}
		s.store.mu.Lock()
		s.store.carts[requestID] = []CartItem{}
		s.store.mu.Unlock()
	}
	http.Redirect(w, r, "/order/"+requestID, http.StatusSeeOther)
}

func (s *Server) HandleOrder(w http.ResponseWriter, r *http.Request) {
	// Извлекаем ID заказа из URL
	path := strings.TrimPrefix(r.URL.Path, "/order/")
	reqID := path
	if reqID == "" {
		reqID = "1"
	}

	// Используем ту же логику, что и в HandleCalc
	var items []CartItem
	var cartMaterials []Material
	var results []Result
	if s.db != nil {
		// Проверяем статус заявки - если rejected, то корзина пустая
		var calculation repository.Calculation
		if err := s.db.Where("id = ?", reqID).First(&calculation).Error; err == nil {
			if calculation.Status == "rejected" {
				// Заявка отклонена, корзина пустая
				items = []CartItem{}
				cartMaterials = []Material{}
			} else {
				// Заявка активна, загружаем товары
				var rs []repository.MaterialCalculation
				_ = s.db.Preload("Material").Where("calculation_id = ?", reqID).Find(&rs).Error
				for _, it := range rs {
					items = append(items, CartItem{MaterialID: it.MaterialID, Quantity: it.Quantity})
					props := buildProps(it.Material)
					imageURL := it.Material.ImageURL
					if !strings.HasPrefix(imageURL, "http") {
						if strings.HasPrefix(imageURL, "/images/") {
							imageURL = "http://localhost:9000" + imageURL
						} else {
							imageURL = s.assetsBase + imageURL
						}
					}
					service := Material{ID: it.Material.ID, Name: it.Material.Name, Description: it.Material.Description, ImageURL: imageURL, Props: props, Comment: it.Comment}
					cartMaterials = append(cartMaterials, service)

					// Если есть сохраненные результаты, добавляем их
					if it.ResultFreq != nil && it.ResultPercent != nil {
						results = append(results, Result{
							MaterialID:   it.MaterialID,
							MaterialName: it.Material.Name,
							NaturalHz:    *it.ResultFreq,
							Isolation:    *it.ResultPercent,
						})
					}
				}
			}
		}
	} else {
		s.store.mu.RLock()
		items = append([]CartItem(nil), s.store.carts[reqID]...)
		s.store.mu.RUnlock()
		for _, it := range items {
			if sv, ok := s.findMaterial(it.MaterialID); ok {
				sv.Comment = it.Comment
				cartMaterials = append(cartMaterials, sv)
			}
		}
	}

	if r.Method == http.MethodPost {
		// Обрабатываем комментарии для товаров (всегда при POST)
		if s.db != nil {
			for _, service := range cartMaterials {
				commentKey := fmt.Sprintf("comment_%d", service.ID)
				comment := r.FormValue(commentKey)
				// Обновляем комментарий в базе данных (даже если пустой)
				_ = s.db.Exec("UPDATE material_calculation SET comment = ? WHERE calculation_id = ? AND material_id = ?",
					comment, reqID, service.ID).Error
			}

			// Перезагружаем данные из базы с обновленными комментариями
			var rs []repository.MaterialCalculation
			_ = s.db.Preload("Material").Where("calculation_id = ?", reqID).Find(&rs).Error
			cartMaterials = []Material{} // Очищаем старые данные
			items = []CartItem{}         // Очищаем старые данные
			for _, it := range rs {
				items = append(items, CartItem{MaterialID: it.MaterialID, Quantity: it.Quantity})
				props := buildProps(it.Material)
				imageURL := it.Material.ImageURL
				if !strings.HasPrefix(imageURL, "http") {
					if strings.HasPrefix(imageURL, "/images/") {
						imageURL = "http://localhost:9000" + imageURL
					} else {
						imageURL = s.assetsBase + imageURL
					}
				}
				service := Material{ID: it.Material.ID, Name: it.Material.Name, Description: it.Material.Description, ImageURL: imageURL, Props: props, Comment: it.Comment}
				cartMaterials = append(cartMaterials, service)
			}
		} else {
			// Обрабатываем комментарии для in-memory store
			s.store.mu.Lock()
			for i := range s.store.carts[reqID] {
				commentKey := fmt.Sprintf("comment_%d", s.store.carts[reqID][i].MaterialID)
				comment := r.FormValue(commentKey)
				s.store.carts[reqID][i].Comment = comment
			}
			s.store.mu.Unlock()

			// Обновляем cartMaterials с новыми комментариями
			cartMaterials = []Material{}
			for _, it := range items {
				if sv, ok := s.findMaterial(it.MaterialID); ok {
					sv.Comment = it.Comment
					cartMaterials = append(cartMaterials, sv)
				}
			}
		}

		// Проверяем, есть ли данные для расчета (масса и частота)
		mass, _ := strconv.ParseFloat(r.FormValue("mass"), 64)
		freq, _ := strconv.ParseFloat(r.FormValue("frequency"), 64)

		if mass > 0 && freq > 0 {
			// Если есть данные для расчета, выполняем расчет
			results = s.calculateResults(items, mass, freq)
			if s.db != nil {
				_ = s.db.Exec("UPDATE calculations SET status = 'completed' WHERE id = ?", reqID).Error

				// Сохраняем результаты расчета в базу данных
				for _, result := range results {
					_ = s.db.Exec("UPDATE material_calculation SET result_freq = ?, result_percent = ? WHERE calculation_id = ? AND material_id = ?",
						result.NaturalHz, result.Isolation, reqID, result.MaterialID).Error
				}
			}
		}
	}

	data := struct {
		Title         string
		RequestID     string
		AssetsBase    string
		CartItems     []CartItem
		CartMaterials []Material
		Results       []Result
		CartCount     int
	}{
		Title:         "Заказ #" + reqID,
		RequestID:     reqID,
		AssetsBase:    s.assetsBase,
		CartItems:     items,
		CartMaterials: cartMaterials,
		Results:       results,
		CartCount:     s.cartCount(reqID),
	}
	if err := s.tmpl.ExecuteTemplate(w, "calc.html", data); err != nil {
		log.Printf("template error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func round(x float64, p int) float64 { pow := math.Pow(10, float64(p)); return math.Round(x*pow) / pow }
