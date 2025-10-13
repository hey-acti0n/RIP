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
// @Summary Удалить материал из расчета
// @Description Удаляет материал из конкретного расчета
// @Tags material-calculations
// @Accept json
// @Produce json
// @Param calculationId path int true "ID расчета"
// @Param materialId path int true "ID материала"
// @Success 200 {object} map[string]string "Материал удален из расчета"
// @Failure 400 {object} map[string]string "Неверные ID"
// @Failure 404 {object} map[string]string "Расчет не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /calculations/{calculationId}/materials/{materialId} [delete]
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
// @Summary Обновить связь расчета и материала
// @Description Обновляет параметры связи между расчетом и материалом
// @Tags material-calculations
// @Accept json
// @Produce json
// @Param calculationId path int true "ID расчета"
// @Param materialId path int true "ID материала"
// @Param request body models.UpdateMaterialCalculationRequest true "Данные для обновления"
// @Success 200 {object} models.MaterialCalculationResponse "Связь обновлена"
// @Failure 400 {object} map[string]string "Неверные данные"
// @Failure 404 {object} map[string]string "Расчет не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /calculations/{calculationId}/materials/{materialId} [put]
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
