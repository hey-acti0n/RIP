package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"rip/internal/app/models"
	"rip/internal/app/service"
)

// GetServices возвращает список услуг с фильтрацией
func (h *ServiceHandler) GetServices(w http.ResponseWriter, r *http.Request) {
	// Парсим параметры фильтрации
	filters := models.ServiceFilters{
		Name:     h.parseQueryParam(r, "name"),
		Material: h.parseQueryParam(r, "material"),
		Page:     1,
		Limit:    10,
	}

	// Парсим числовые параметры
	if thicknessMinStr := h.parseQueryParam(r, "thickness_min"); thicknessMinStr != "" {
		if val, err := strconv.ParseFloat(thicknessMinStr, 64); err == nil {
			filters.ThicknessMin = &val
		}
	}
	if thicknessMaxStr := h.parseQueryParam(r, "thickness_max"); thicknessMaxStr != "" {
		if val, err := strconv.ParseFloat(thicknessMaxStr, 64); err == nil {
			filters.ThicknessMax = &val
		}
	}
	if densityMinStr := h.parseQueryParam(r, "density_min"); densityMinStr != "" {
		if val, err := strconv.ParseFloat(densityMinStr, 64); err == nil {
			filters.DensityMin = &val
		}
	}
	if densityMaxStr := h.parseQueryParam(r, "density_max"); densityMaxStr != "" {
		if val, err := strconv.ParseFloat(densityMaxStr, 64); err == nil {
			filters.DensityMax = &val
		}
	}

	// Парсим пагинацию
	page, limit := h.parsePagination(r)
	filters.Page = page
	filters.Limit = limit

	// Получаем данные
	services, total, err := h.service.ServiceService.GetServices(r.Context(), filters)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Ошибка получения услуг")
		return
	}

	// Конвертируем в ответ
	var serviceResponses []models.ServiceResponse
	for _, service := range services {
		serviceResponses = append(serviceResponses, models.ConvertToServiceResponse(service))
	}

	// Формируем ответ с пагинацией
	response := PaginationResponse{
		Data: serviceResponses,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      int(total),
			TotalPages: calculateTotalPages(int(total), limit),
		},
	}

	h.writeJSON(w, http.StatusOK, response)
}

// GetService возвращает одну услугу
func (h *ServiceHandler) GetService(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID услуги")
		return
	}

	svc, err := h.service.ServiceService.GetService(r.Context(), id)
	if err != nil {
		if err == service.ErrServiceNotFound {
			h.writeError(w, http.StatusNotFound, "Услуга не найдена")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка получения услуги")
		return
	}

	response := models.ConvertToServiceResponse(*svc)
	h.writeJSON(w, http.StatusOK, response)
}

// CreateService создает новую услугу
func (h *ServiceHandler) CreateService(w http.ResponseWriter, r *http.Request) {
	var req models.CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}

	// Валидация
	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "Название услуги обязательно")
		return
	}

	svc, err := h.service.ServiceService.CreateService(r.Context(), req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Ошибка создания услуги")
		return
	}

	response := models.ConvertToServiceResponse(*svc)
	h.writeJSON(w, http.StatusCreated, response)
}

// UpdateService обновляет услугу
func (h *ServiceHandler) UpdateService(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID услуги")
		return
	}

	var req models.UpdateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}

	svc, err := h.service.ServiceService.UpdateService(r.Context(), id, req)
	if err != nil {
		if err == service.ErrServiceNotFound {
			h.writeError(w, http.StatusNotFound, "Услуга не найдена")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка обновления услуги")
		return
	}

	response := models.ConvertToServiceResponse(*svc)
	h.writeJSON(w, http.StatusOK, response)
}

// DeleteService удаляет услугу
func (h *ServiceHandler) DeleteService(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID услуги")
		return
	}

	err = h.service.ServiceService.DeleteService(r.Context(), id)
	if err != nil {
		if err == service.ErrServiceNotFound {
			h.writeError(w, http.StatusNotFound, "Услуга не найдена")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка удаления услуги")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Услуга удалена"})
}

// AddServiceToCart добавляет услугу в корзину
func (h *ServiceHandler) AddServiceToCart(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID услуги")
		return
	}

	request, err := h.service.ServiceService.AddServiceToCart(r.Context(), id)
	if err != nil {
		if err == service.ErrServiceNotFound {
			h.writeError(w, http.StatusNotFound, "Услуга не найдена")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка добавления в корзину")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "Услуга добавлена в корзину",
		"request_id": request.ID,
	})
}

// UploadServiceImage загружает изображение для услуги
func (h *ServiceHandler) UploadServiceImage(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID услуги")
		return
	}

	// Проверяем существование услуги
	_, err = h.service.ServiceService.GetService(r.Context(), id)
	if err != nil {
		if err == service.ErrServiceNotFound {
			h.writeError(w, http.StatusNotFound, "Услуга не найдена")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка проверки услуги")
		return
	}

	// Парсим multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB
		h.writeError(w, http.StatusBadRequest, "Ошибка парсинга формы")
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Файл изображения не найден")
		return
	}
	defer file.Close()

	// TODO: Загрузить в Minio и обновить ImageURL
	// Пока возвращаем заглушку
	h.writeJSON(w, http.StatusOK, map[string]string{
		"message":   "Изображение загружено",
		"image_url": "/images/service_" + strconv.Itoa(id) + ".jpg",
	})
}
