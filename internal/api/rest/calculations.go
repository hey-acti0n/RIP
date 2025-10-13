package rest

import (
	"encoding/json"
	"net/http"

	"rip/internal/app/models"
	"rip/internal/app/service"
)

// GetCartInfo возвращает информацию о корзине
// @Summary Получить информацию о корзине
// @Description Возвращает общую информацию о корзине (количество товаров, общая стоимость)
// @Tags calculations
// @Accept json
// @Produce json
// @Success 200 {object} models.CartInfo "Информация о корзине"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /calculations/cart-info [get]
func (h *CalculationHandler) GetCartInfo(w http.ResponseWriter, r *http.Request) {
	cartInfo, err := h.service.CalculationService.GetCartInfo(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Ошибка получения информации о корзине")
		return
	}

	h.writeJSON(w, http.StatusOK, cartInfo)
}

// GetCalculations возвращает список расчётов с фильтрацией
// @Summary Получить список расчетов
// @Description Возвращает список расчетов с возможностью фильтрации. Для аутентифицированных пользователей показывает только их расчеты, для модераторов - все расчеты
// @Tags calculations
// @Accept json
// @Produce json
// @Param status query string false "Статус расчета"
// @Param formed_from query string false "Дата начала (YYYY-MM-DD)"
// @Param formed_to query string false "Дата окончания (YYYY-MM-DD)"
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество записей на странице" default(10)
// @Success 200 {object} PaginationResponse "Список расчетов"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /calculations [get]
func (h *CalculationHandler) GetCalculations(w http.ResponseWriter, r *http.Request) {
	// Парсим параметры фильтрации
	filters := models.CalculationFilters{
		Status: h.parseQueryParam(r, "status"),
		Page:   1,
		Limit:  10,
	}

	// Парсим диапазон дат
	formedFrom, formedTo := h.parseDateRange(r)
	filters.FormedFrom = formedFrom
	filters.FormedTo = formedTo

	// Парсим пагинацию
	page, limit := h.parsePagination(r)
	filters.Page = page
	filters.Limit = limit

	// Проверяем аутентификацию и устанавливаем фильтр по пользователю
	userID, isAuthenticated := r.Context().Value("user_id").(int)
	if isAuthenticated {
		// Если пользователь аутентифицирован, показываем только его расчеты
		filters.CreatorID = &userID
	}

	// Получаем данные
	calculations, total, err := h.service.CalculationService.GetCalculations(r.Context(), filters)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Ошибка получения расчётов")
		return
	}

	// Конвертируем в ответ
	var calculationResponses []models.CalculationResponse
	for _, calc := range calculations {
		calculationResponses = append(calculationResponses, models.ConvertToCalculationResponse(calc))
	}

	// Формируем ответ с пагинацией
	response := PaginationResponse{
		Data: calculationResponses,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      int(total),
			TotalPages: calculateTotalPages(int(total), limit),
		},
	}

	h.writeJSON(w, http.StatusOK, response)
}

// GetCalculation возвращает один расчёт с материалами
// @Summary Получить расчет по ID
// @Description Возвращает информацию о конкретном расчете по его идентификатору
// @Tags calculations
// @Accept json
// @Produce json
// @Param id path int true "ID расчета"
// @Success 200 {object} models.CalculationResponse "Информация о расчете"
// @Failure 400 {object} map[string]string "Неверный ID"
// @Failure 404 {object} map[string]string "Расчет не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /calculations/{id} [get]
func (h *CalculationHandler) GetCalculation(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID расчёта")
		return
	}

	calculation, err := h.service.CalculationService.GetCalculation(r.Context(), id)
	if err != nil {
		if err == service.ErrCalculationNotFound {
			h.writeError(w, http.StatusNotFound, "Расчёт не найден")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка получения расчёта")
		return
	}

	// Конвертируем материалы
	var materialCalculationResponses []models.MaterialCalculationResponse
	for _, mc := range calculation.MaterialCalculations {
		materialCalculationResponses = append(materialCalculationResponses, models.ConvertToMaterialCalculationResponse(mc))
	}

	// Формируем полный ответ
	response := struct {
		models.CalculationResponse
		MaterialCalculations []models.MaterialCalculationResponse `json:"material_calculations"`
	}{
		CalculationResponse: models.CalculationResponse{
			ID:             calculation.ID,
			Status:         calculation.Status,
			Title:          calculation.Title,
			Description:    calculation.Description,
			CreatorID:      calculation.CreatorID,
			CreatorLogin:   calculation.CreatorLogin,
			ModeratorID:    calculation.ModeratorID,
			ModeratorLogin: calculation.ModeratorLogin,
			CreatedAt:      calculation.CreatedAt,
			FormedAt:       calculation.FormedAt,
			CompletedAt:    calculation.CompletedAt,
			TotalCost:      calculation.TotalCost,
			DeliveryDate:   calculation.DeliveryDate,
		},
		MaterialCalculations: materialCalculationResponses,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// GetCalculationMaterials возвращает список материалов в расчёте
// @Summary Получить материалы расчета
// @Description Возвращает список материалов, связанных с конкретным расчетом
// @Tags calculations
// @Accept json
// @Produce json
// @Param id path int true "ID расчета"
// @Success 200 {array} models.MaterialCalculationResponse "Список материалов расчета"
// @Failure 400 {object} map[string]string "Неверный ID"
// @Failure 404 {object} map[string]string "Расчет не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /calculations/{id}/materials [get]
func (h *CalculationHandler) GetCalculationMaterials(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID расчёта")
		return
	}

	calculation, err := h.service.CalculationService.GetCalculation(r.Context(), id)
	if err != nil {
		if err == service.ErrCalculationNotFound {
			h.writeError(w, http.StatusNotFound, "Расчёт не найден")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка получения расчёта")
		return
	}

	// Конвертируем материалы
	var materialCalculationResponses []models.MaterialCalculationResponse
	for _, mc := range calculation.MaterialCalculations {
		materialCalculationResponses = append(materialCalculationResponses, models.ConvertToMaterialCalculationResponse(mc))
	}

	h.writeJSON(w, http.StatusOK, materialCalculationResponses)
}

// UpdateCalculation обновляет поля расчёта
// @Summary Обновить расчет
// @Description Обновляет информацию о существующем расчете
// @Tags calculations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID расчета"
// @Param request body models.UpdateCalculationRequest true "Данные для обновления"
// @Success 200 {object} models.CalculationResponse "Расчет обновлен"
// @Failure 400 {object} map[string]string "Неверные данные"
// @Failure 401 {object} map[string]string "Пользователь не аутентифицирован"
// @Failure 404 {object} map[string]string "Расчет не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /calculations/{id} [put]
func (h *CalculationHandler) UpdateCalculation(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID расчёта")
		return
	}

	var req models.UpdateCalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}

	err = h.service.CalculationService.UpdateCalculation(r.Context(), id, req)
	if err != nil {
		if err == service.ErrCalculationNotFound {
			h.writeError(w, http.StatusNotFound, "Расчёт не найден")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка обновления расчёта")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Расчёт обновлён"})
}

// FormCalculation формирует расчёт
// @Summary Сформировать расчет
// @Description Формирует расчет для дальнейшего завершения или отклонения
// @Tags calculations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID расчета"
// @Success 200 {object} map[string]string "Расчет сформирован"
// @Failure 400 {object} map[string]string "Неверные данные"
// @Failure 401 {object} map[string]string "Пользователь не аутентифицирован"
// @Failure 404 {object} map[string]string "Расчет не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /calculations/{id}/form [put]
func (h *CalculationHandler) FormCalculation(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID расчёта")
		return
	}

	err = h.service.CalculationService.FormCalculation(r.Context(), id)
	if err != nil {
		if err == service.ErrCalculationNotFound {
			h.writeError(w, http.StatusNotFound, "Расчёт не найден")
			return
		}
		if err == service.ErrCalculationMissingRequiredFields {
			h.writeError(w, http.StatusBadRequest, "Отсутствуют обязательные поля расчёта")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка формирования расчёта")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Расчёт сформирован"})
}

// CompleteCalculation завершает или отклоняет расчёт
// @Summary Завершить или отклонить расчет
// @Description Завершает или отклоняет расчет. Доступно только модераторам
// @Tags calculations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID расчета"
// @Param action query string true "Действие: complete или reject"
// @Success 200 {object} models.CompleteCalculationResponse "Результат операции"
// @Failure 400 {object} map[string]string "Неверные данные"
// @Failure 401 {object} map[string]string "Пользователь не аутентифицирован"
// @Failure 403 {object} map[string]string "Недостаточно прав"
// @Failure 404 {object} map[string]string "Расчет не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /calculations/{id}/status [put]
func (h *CalculationHandler) CompleteCalculation(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID расчёта")
		return
	}

	// Получаем action из query параметра (как в Postman коллекции)
	action := h.parseQueryParam(r, "action")
	if action == "" {
		h.writeError(w, http.StatusBadRequest, "Параметр action обязателен")
		return
	}

	if action != "complete" && action != "reject" {
		h.writeError(w, http.StatusBadRequest, "Действие должно быть 'complete' или 'reject'")
		return
	}

	// Получаем ID модератора из контекста
	moderatorID, ok := r.Context().Value("user_id").(int)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Модератор не аутентифицирован")
		return
	}

	response, err := h.service.CalculationService.CompleteCalculation(r.Context(), id, action, moderatorID)
	if err != nil {
		if err == service.ErrCalculationNotFound {
			h.writeError(w, http.StatusNotFound, "Расчёт не найден")
			return
		}
		if err == service.ErrInvalidAction {
			h.writeError(w, http.StatusBadRequest, "Неверное действие")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка обработки расчёта")
		return
	}

	h.writeJSON(w, http.StatusOK, response)
}

// DeleteCalculation удаляет расчёт
// @Summary Удалить расчет
// @Description Удаляет расчет из системы. Доступно только модераторам
// @Tags calculations
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "ID расчета"
// @Success 204 "Расчет удален"
// @Failure 400 {object} map[string]string "Неверный ID"
// @Failure 401 {object} map[string]string "Пользователь не аутентифицирован"
// @Failure 403 {object} map[string]string "Недостаточно прав"
// @Failure 404 {object} map[string]string "Расчет не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /calculations/{id} [delete]
func (h *CalculationHandler) DeleteCalculation(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID расчёта")
		return
	}

	err = h.service.CalculationService.DeleteCalculation(r.Context(), id)
	if err != nil {
		if err == service.ErrCalculationNotFound {
			h.writeError(w, http.StatusNotFound, "Расчёт не найден")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка удаления расчёта")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Расчёт удалён"})
}
