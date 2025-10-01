package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"rip/internal/app/models"
	"rip/internal/app/service"
)

// GetMaterials возвращает список материалов с фильтрацией
func (h *MaterialHandler) GetMaterials(w http.ResponseWriter, r *http.Request) {
	// Парсим параметры фильтрации
	filters := models.MaterialFilters{
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
	materials, total, err := h.service.MaterialService.GetMaterials(r.Context(), filters)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Ошибка получения материалов")
		return
	}

	// Конвертируем в ответ
	var materialResponses []models.MaterialResponse
	for _, material := range materials {
		materialResponses = append(materialResponses, models.ConvertToMaterialResponse(material))
	}

	// Формируем ответ с пагинацией
	response := PaginationResponse{
		Data: materialResponses,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      int(total),
			TotalPages: calculateTotalPages(int(total), limit),
		},
	}

	h.writeJSON(w, http.StatusOK, response)
}

// GetMaterial возвращает один материал
func (h *MaterialHandler) GetMaterial(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID материала")
		return
	}

	material, err := h.service.MaterialService.GetMaterial(r.Context(), id)
	if err != nil {
		if err == service.ErrMaterialNotFound {
			h.writeError(w, http.StatusNotFound, "Материал не найден")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка получения материала")
		return
	}

	response := models.ConvertToMaterialResponse(*material)
	h.writeJSON(w, http.StatusOK, response)
}

// CreateMaterial создает новый материал
func (h *MaterialHandler) CreateMaterial(w http.ResponseWriter, r *http.Request) {
	var req models.CreateMaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}

	// Валидация
	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "Название материала обязательно")
		return
	}

	material, err := h.service.MaterialService.CreateMaterial(r.Context(), req)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Ошибка создания материала")
		return
	}

	response := models.ConvertToMaterialResponse(*material)
	h.writeJSON(w, http.StatusCreated, response)
}

// UpdateMaterial обновляет материал
func (h *MaterialHandler) UpdateMaterial(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID материала")
		return
	}

	var req models.UpdateMaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}

	material, err := h.service.MaterialService.UpdateMaterial(r.Context(), id, req)
	if err != nil {
		if err == service.ErrMaterialNotFound {
			h.writeError(w, http.StatusNotFound, "Материал не найден")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка обновления материала")
		return
	}

	response := models.ConvertToMaterialResponse(*material)
	h.writeJSON(w, http.StatusOK, response)
}

// DeleteMaterial удаляет материал
func (h *MaterialHandler) DeleteMaterial(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID материала")
		return
	}

	err = h.service.MaterialService.DeleteMaterial(r.Context(), id)
	if err != nil {
		if err == service.ErrMaterialNotFound {
			h.writeError(w, http.StatusNotFound, "Материал не найден")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка удаления материала")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Материал удален"})
}

// AddMaterialToCart добавляет материал в корзину
func (h *MaterialHandler) AddMaterialToCart(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID материала")
		return
	}

	calculation, err := h.service.MaterialService.AddMaterialToCart(r.Context(), id)
	if err != nil {
		if err == service.ErrMaterialNotFound {
			h.writeError(w, http.StatusNotFound, "Материал не найден")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка добавления в корзину")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":        "Материал добавлен в корзину",
		"calculation_id": calculation.ID,
	})
}

// UploadMaterialImage загружает изображение для материала
func (h *MaterialHandler) UploadMaterialImage(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID материала")
		return
	}

	// Проверяем существование материала
	_, err = h.service.MaterialService.GetMaterial(r.Context(), id)
	if err != nil {
		if err == service.ErrMaterialNotFound {
			h.writeError(w, http.StatusNotFound, "Материал не найден")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка проверки материала")
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
