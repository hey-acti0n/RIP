package rest

import (
	"encoding/json"
	"net/http"

	"rip/internal/app/models"
	"rip/internal/app/service"
)

// GetCartInfo возвращает информацию о корзине
func (h *RequestHandler) GetCartInfo(w http.ResponseWriter, r *http.Request) {
	cartInfo, err := h.service.RequestService.GetCartInfo(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Ошибка получения информации о корзине")
		return
	}

	h.writeJSON(w, http.StatusOK, cartInfo)
}

// GetRequests возвращает список заявок с фильтрацией
func (h *RequestHandler) GetRequests(w http.ResponseWriter, r *http.Request) {
	// Парсим параметры фильтрации
	filters := models.RequestFilters{
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
	requests, total, err := h.service.RequestService.GetRequests(r.Context(), filters)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "Ошибка получения заявок")
		return
	}

	// Конвертируем в ответ
	var requestResponses []models.RequestResponse
	for _, req := range requests {
		requestResponses = append(requestResponses, models.ConvertToRequestResponse(req))
	}

	// Формируем ответ с пагинацией
	response := PaginationResponse{
		Data: requestResponses,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      int(total),
			TotalPages: calculateTotalPages(int(total), limit),
		},
	}

	h.writeJSON(w, http.StatusOK, response)
}

// GetRequest возвращает одну заявку с услугами
func (h *RequestHandler) GetRequest(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID заявки")
		return
	}

	request, err := h.service.RequestService.GetRequest(r.Context(), id)
	if err != nil {
		if err == service.ErrRequestNotFound {
			h.writeError(w, http.StatusNotFound, "Заявка не найдена")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка получения заявки")
		return
	}

	// Конвертируем услуги
	var requestServiceResponses []models.RequestServiceResponse
	for _, rs := range request.RequestServices {
		requestServiceResponses = append(requestServiceResponses, models.ConvertToRequestServiceResponse(rs))
	}

	// Формируем полный ответ
	response := struct {
		models.RequestResponse
		RequestServices []models.RequestServiceResponse `json:"request_services"`
	}{
		RequestResponse: models.RequestResponse{
			ID:             request.ID,
			Status:         request.Status,
			Title:          request.Title,
			Description:    request.Description,
			CreatorID:      request.CreatorID,
			CreatorLogin:   request.CreatorLogin,
			ModeratorID:    request.ModeratorID,
			ModeratorLogin: request.ModeratorLogin,
			CreatedAt:      request.CreatedAt,
			FormedAt:       request.FormedAt,
			CompletedAt:    request.CompletedAt,
			TotalCost:      request.TotalCost,
			DeliveryDate:   request.DeliveryDate,
		},
		RequestServices: requestServiceResponses,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// UpdateRequest обновляет поля заявки
func (h *RequestHandler) UpdateRequest(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID заявки")
		return
	}

	var req models.UpdateRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}

	err = h.service.RequestService.UpdateRequest(r.Context(), id, req)
	if err != nil {
		if err == service.ErrRequestNotFound {
			h.writeError(w, http.StatusNotFound, "Заявка не найдена")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка обновления заявки")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Заявка обновлена"})
}

// FormRequest формирует заявку
func (h *RequestHandler) FormRequest(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID заявки")
		return
	}

	err = h.service.RequestService.FormRequest(r.Context(), id)
	if err != nil {
		if err == service.ErrRequestNotFound {
			h.writeError(w, http.StatusNotFound, "Заявка не найдена")
			return
		}
		if err == service.ErrRequestMissingRequiredFields {
			h.writeError(w, http.StatusBadRequest, "Отсутствуют обязательные поля заявки")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка формирования заявки")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Заявка сформирована"})
}

// CompleteRequest завершает или отклоняет заявку
func (h *RequestHandler) CompleteRequest(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID заявки")
		return
	}

	var req struct {
		Action string `json:"action" validate:"required,oneof=complete reject"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}

	if req.Action != "complete" && req.Action != "reject" {
		h.writeError(w, http.StatusBadRequest, "Действие должно быть 'complete' или 'reject'")
		return
	}

	err = h.service.RequestService.CompleteRequest(r.Context(), id, req.Action)
	if err != nil {
		if err == service.ErrRequestNotFound {
			h.writeError(w, http.StatusNotFound, "Заявка не найдена")
			return
		}
		if err == service.ErrInvalidAction {
			h.writeError(w, http.StatusBadRequest, "Неверное действие")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка обработки заявки")
		return
	}

	message := "Заявка завершена"
	if req.Action == "reject" {
		message = "Заявка отклонена"
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": message})
}

// DeleteRequest удаляет заявку
func (h *RequestHandler) DeleteRequest(w http.ResponseWriter, r *http.Request) {
	id, err := h.parseID(r)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID заявки")
		return
	}

	err = h.service.RequestService.DeleteRequest(r.Context(), id)
	if err != nil {
		if err == service.ErrRequestNotFound {
			h.writeError(w, http.StatusNotFound, "Заявка не найдена")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка удаления заявки")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Заявка удалена"})
}
