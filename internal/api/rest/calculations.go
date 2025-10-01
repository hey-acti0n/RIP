package rest

import (
	"encoding/json"
	"net/http"

	"rip/internal/app/models"
	"rip/internal/app/service"
)

// GetCartInfo возвращает информацию о корзине
func (h *CalculationHandler) GetCartInfo(w http.ResponseWriter, r *http.Request) {
	cartInfo, err := h.service.CalculationService.GetCartInfo(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Ошибка получения информации о корзине")
		return
	}

	h.writeJSON(w, http.StatusOK, cartInfo)
}

// GetCalculations возвращает список расчётов с фильтрацией
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

	response, err := h.service.CalculationService.CompleteCalculation(r.Context(), id, action)
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
