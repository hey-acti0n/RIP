package rest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"rip/internal/app/service"

	"github.com/gorilla/mux"
)

// BaseHandler содержит общие методы для всех обработчиков
type BaseHandler struct {
	service *service.Service
}

// NewBaseHandler создает новый базовый обработчик
func NewBaseHandler(svc *service.Service) *BaseHandler {
	return &BaseHandler{service: svc}
}

// writeJSON отправляет JSON ответ
func (h *BaseHandler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
}

// writeError отправляет ошибку в JSON формате
func (h *BaseHandler) writeError(w http.ResponseWriter, statusCode int, message string) {
	h.writeJSON(w, statusCode, map[string]string{"error": message})
}

// parseID извлекает ID из URL параметров
func (h *BaseHandler) parseID(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		return 0, ErrInvalidID
	}
	return strconv.Atoi(idStr)
}

// parseQueryParam извлекает параметр из query string
func (h *BaseHandler) parseQueryParam(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}

// parsePagination извлекает параметры пагинации
func (h *BaseHandler) parsePagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(h.parseQueryParam(r, "page"))
	limit, _ := strconv.Atoi(h.parseQueryParam(r, "limit"))

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	return page, limit
}

// parseDateRange извлекает диапазон дат из query параметров
func (h *BaseHandler) parseDateRange(r *http.Request) (*time.Time, *time.Time) {
	fromStr := h.parseQueryParam(r, "formed_from")
	toStr := h.parseQueryParam(r, "formed_to")

	var fromPtr, toPtr *time.Time
	if fromStr != "" {
		if from, err := time.Parse("2006-01-02", fromStr); err == nil {
			fromPtr = &from
		}
	}
	if toStr != "" {
		if to, err := time.Parse("2006-01-02", toStr); err == nil {
			toPtr = &to
		}
	}

	return fromPtr, toPtr
}

// ServiceHandler обрабатывает запросы к услугам
type ServiceHandler struct {
	*BaseHandler
}

// NewServiceHandler создает новый обработчик услуг
func NewServiceHandler(svc *service.Service) *ServiceHandler {
	return &ServiceHandler{BaseHandler: NewBaseHandler(svc)}
}

// RequestHandler обрабатывает запросы к заявкам
type RequestHandler struct {
	*BaseHandler
}

// NewRequestHandler создает новый обработчик заявок
func NewRequestHandler(svc *service.Service) *RequestHandler {
	return &RequestHandler{BaseHandler: NewBaseHandler(svc)}
}

// RequestServiceHandler обрабатывает запросы к связи заявка-услуга
type RequestServiceHandler struct {
	*BaseHandler
}

// NewRequestServiceHandler создает новый обработчик связи заявка-услуга
func NewRequestServiceHandler(svc *service.Service) *RequestServiceHandler {
	return &RequestServiceHandler{BaseHandler: NewBaseHandler(svc)}
}

// UserHandler обрабатывает запросы к пользователям
type UserHandler struct {
	*BaseHandler
}

// NewUserHandler создает новый обработчик пользователей
func NewUserHandler(svc *service.Service) *UserHandler {
	return &UserHandler{BaseHandler: NewBaseHandler(svc)}
}

// Ошибки API
var (
	ErrInvalidID      = &APIError{Code: "INVALID_ID", Message: "Неверный ID", StatusCode: http.StatusBadRequest}
	ErrNotFound       = &APIError{Code: "NOT_FOUND", Message: "Ресурс не найден", StatusCode: http.StatusNotFound}
	ErrInvalidRequest = &APIError{Code: "INVALID_REQUEST", Message: "Неверный запрос", StatusCode: http.StatusBadRequest}
	ErrUnauthorized   = &APIError{Code: "UNAUTHORIZED", Message: "Не авторизован", StatusCode: http.StatusUnauthorized}
	ErrForbidden      = &APIError{Code: "FORBIDDEN", Message: "Доступ запрещен", StatusCode: http.StatusForbidden}
	ErrInternalError  = &APIError{Code: "INTERNAL_ERROR", Message: "Внутренняя ошибка сервера", StatusCode: http.StatusInternalServerError}
)

// APIError представляет ошибку API
type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

func (e *APIError) Error() string {
	return e.Message
}

// ErrorResponse представляет ответ с ошибкой
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// PaginationResponse представляет ответ с пагинацией
type PaginationResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination содержит информацию о пагинации
type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// calculateTotalPages вычисляет общее количество страниц
func calculateTotalPages(total, limit int) int {
	if limit <= 0 {
		return 1
	}
	pages := total / limit
	if total%limit > 0 {
		pages++
	}
	if pages == 0 {
		pages = 1
	}
	return pages
}

// MaterialHandler обрабатывает запросы для материалов
type MaterialHandler struct {
	*BaseHandler
}

// NewMaterialHandler создает новый обработчик материалов
func NewMaterialHandler(svc *service.Service) *MaterialHandler {
	return &MaterialHandler{BaseHandler: NewBaseHandler(svc)}
}

// CalculationHandler обрабатывает запросы для расчётов
type CalculationHandler struct {
	*BaseHandler
}

// NewCalculationHandler создает новый обработчик расчётов
func NewCalculationHandler(svc *service.Service) *CalculationHandler {
	return &CalculationHandler{BaseHandler: NewBaseHandler(svc)}
}

// MaterialCalculationHandler обрабатывает запросы для связи расчёт-материал
type MaterialCalculationHandler struct {
	*BaseHandler
}

// NewMaterialCalculationHandler создает новый обработчик связи расчёт-материал
func NewMaterialCalculationHandler(svc *service.Service) *MaterialCalculationHandler {
	return &MaterialCalculationHandler{BaseHandler: NewBaseHandler(svc)}
}
