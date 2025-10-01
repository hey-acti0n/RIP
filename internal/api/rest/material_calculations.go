package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"rip/internal/app/models"
	"rip/internal/app/service"

	"github.com/gorilla/mux"
)

// DeleteMaterialCalculation удаляет материал из расчёта
func (h *MaterialCalculationHandler) DeleteMaterialCalculation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	calculationID, err := strconv.Atoi(vars["calculationId"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID расчёта")
		return
	}

	materialID, err := strconv.Atoi(vars["materialId"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID материала")
		return
	}

	err = h.service.MaterialCalculationService.DeleteMaterialCalculation(r.Context(), calculationID, materialID)
	if err != nil {
		if err == service.ErrCalculationNotFound {
			h.writeError(w, http.StatusNotFound, "Расчёт не найден")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка удаления материала из расчёта")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Материал удалён из расчёта"})
}

// UpdateMaterialCalculation обновляет связь расчёт-материал
func (h *MaterialCalculationHandler) UpdateMaterialCalculation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	calculationID, err := strconv.Atoi(vars["calculationId"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID расчёта")
		return
	}

	materialID, err := strconv.Atoi(vars["materialId"])
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный ID материала")
		return
	}

	var req models.UpdateMaterialCalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}

	err = h.service.MaterialCalculationService.UpdateMaterialCalculation(r.Context(), calculationID, materialID, req)
	if err != nil {
		if err == service.ErrCalculationNotFound {
			h.writeError(w, http.StatusNotFound, "Расчёт не найден")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка обновления связи расчёт-материал")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Связь расчёт-материал обновлена"})
}
