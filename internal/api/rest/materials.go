package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"rip/internal/app/models"
	"rip/internal/app/service"
)

// GetMaterials возвращает список материалов с фильтрацией
// @Summary Получить список материалов
// @Description Возвращает список материалов с возможностью фильтрации по различным параметрам
// @Tags materials
// @Accept json
// @Produce json
// @Param name query string false "Поиск по названию"
// @Param material query string false "Фильтр по материалу"
// @Param thickness_min query number false "Минимальная толщина"
// @Param thickness_max query number false "Максимальная толщина"
// @Param density_min query number false "Минимальная плотность"
// @Param density_max query number false "Максимальная плотность"
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество записей на странице" default(10)
// @Success 200 {object} PaginationResponse "Список материалов"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /materials [get]
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
// @Summary Получить материал по ID
// @Description Возвращает информацию о конкретном материале по его идентификатору
// @Tags materials
// @Accept json
// @Produce json
// @Param id path int true "ID материала"
// @Success 200 {object} models.MaterialResponse "Информация о материале"
// @Failure 400 {object} map[string]string "Неверный ID"
// @Failure 404 {object} map[string]string "Материал не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /materials/{id} [get]
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
// @Summary Создать новый материал
// @Description Создает новый материал в системе
// @Tags materials
// @Accept json
// @Produce json
// @Param request body models.CreateMaterialRequest true "Данные для создания материала"
// @Success 201 {object} models.MaterialResponse "Материал создан"
// @Failure 400 {object} map[string]string "Неверные данные"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /materials [post]
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
// @Summary Обновить материал
// @Description Обновляет информацию о существующем материале
// @Tags materials
// @Accept json
// @Produce json
// @Param id path int true "ID материала"
// @Param request body models.UpdateMaterialRequest true "Данные для обновления"
// @Success 200 {object} models.MaterialResponse "Материал обновлен"
// @Failure 400 {object} map[string]string "Неверные данные"
// @Failure 404 {object} map[string]string "Материал не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /materials/{id} [put]
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
// @Summary Удалить материал
// @Description Удаляет материал из системы
// @Tags materials
// @Accept json
// @Produce json
// @Param id path int true "ID материала"
// @Success 204 "Материал удален"
// @Failure 400 {object} map[string]string "Неверный ID"
// @Failure 404 {object} map[string]string "Материал не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /materials/{id} [delete]
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
// @Summary Добавить материал в корзину
// @Description Добавляет материал в корзину для последующего расчета
// @Tags materials
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID материала"
// @Success 200 {object} map[string]string "Материал добавлен в корзину"
// @Failure 400 {object} map[string]string "Неверный ID"
// @Failure 404 {object} map[string]string "Материал не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /materials/{id}/add-to-cart [post]
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
// @Summary Загрузить изображение материала
// @Description Загружает изображение для материала
// @Tags materials
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID материала"
// @Param image formData file true "Изображение материала"
// @Success 200 {object} map[string]string "Изображение загружено"
// @Failure 400 {object} map[string]string "Неверные данные"
// @Failure 404 {object} map[string]string "Материал не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /materials/{id}/image [post]
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
