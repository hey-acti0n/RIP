package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"rip/internal/app/models"
	"rip/internal/app/service"

	"github.com/gorilla/mux"
)

// DeleteRequestService удаляет услугу из заявки
func (h *RequestServiceHandler) DeleteRequestService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	requestID, err := strconv.Atoi(vars["requestId"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID заявки")
		return
	}

	serviceID, err := strconv.Atoi(vars["serviceId"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID услуги")
		return
	}

	err = h.service.RequestServiceService.DeleteRequestService(r.Context(), requestID, serviceID)
	if err != nil {
		if err == service.ErrRequestNotFound {
			h.writeError(w, http.StatusNotFound, "Заявка не найдена")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка удаления услуги из заявки")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Услуга удалена из заявки"})
}

// UpdateRequestService обновляет связь заявка-услуга
func (h *RequestServiceHandler) UpdateRequestService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	requestID, err := strconv.Atoi(vars["requestId"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID заявки")
		return
	}

	serviceID, err := strconv.Atoi(vars["serviceId"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID услуги")
		return
	}

	var req models.UpdateRequestServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}

	err = h.service.RequestServiceService.UpdateRequestService(r.Context(), requestID, serviceID, req)
	if err != nil {
		if err == service.ErrRequestNotFound {
			h.writeError(w, http.StatusNotFound, "Заявка не найдена")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка обновления связи заявка-услуга")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Связь заявка-услуга обновлена"})
}
