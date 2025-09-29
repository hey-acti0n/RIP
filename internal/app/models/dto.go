package models

import (
	"time"

	"rip/internal/app/repository"
)

// ServiceFilters содержит фильтры для поиска услуг
type ServiceFilters struct {
	Name         string   `json:"name"`
	Material     string   `json:"material"`
	ThicknessMin *float64 `json:"thickness_min"`
	ThicknessMax *float64 `json:"thickness_max"`
	DensityMin   *float64 `json:"density_min"`
	DensityMax   *float64 `json:"density_max"`
	Page         int      `json:"page"`
	Limit        int      `json:"limit"`
}

// CreateServiceRequest содержит данные для создания услуги
type CreateServiceRequest struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description"`
	ImageURL    string   `json:"image_url"`
	Density     *float64 `json:"density"`
	Thickness   *float64 `json:"thickness"`
	Material    *string  `json:"material"`
}

// UpdateServiceRequest содержит данные для обновления услуги
type UpdateServiceRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ImageURL    string   `json:"image_url"`
	Density     *float64 `json:"density"`
	Thickness   *float64 `json:"thickness"`
	Material    *string  `json:"material"`
}

// RequestFilters содержит фильтры для поиска заявок
type RequestFilters struct {
	Status     string     `json:"status"`
	FormedFrom *time.Time `json:"formed_from"`
	FormedTo   *time.Time `json:"formed_to"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
}

// RequestWithUsers содержит заявку с информацией о пользователях
type RequestWithUsers struct {
	repository.Request
	CreatorLogin   string  `json:"creator_login"`
	ModeratorLogin *string `json:"moderator_login,omitempty"`
}

// RequestWithServices содержит заявку с услугами
type RequestWithServices struct {
	repository.Request
	CreatorLogin    string                      `json:"creator_login"`
	ModeratorLogin  *string                     `json:"moderator_login,omitempty"`
	RequestServices []repository.RequestService `json:"request_services"`
}

// UpdateRequestRequest содержит данные для обновления заявки
type UpdateRequestRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// CartInfo содержит информацию о корзине
type CartInfo struct {
	RequestID int `json:"request_id"`
	ItemCount int `json:"item_count"`
}

// UpdateRequestServiceRequest содержит данные для обновления связи заявка-услуга
type UpdateRequestServiceRequest struct {
	Quantity  *int   `json:"quantity"`
	SortOrder *int   `json:"sort_order"`
	IsMain    *bool  `json:"is_main"`
	Comment   string `json:"comment"`
}

// RegisterRequest содержит данные для регистрации пользователя
type RegisterRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	FullName string `json:"full_name" validate:"required"`
}

// LoginRequest содержит данные для входа
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse содержит ответ при входе
type LoginResponse struct {
	User  repository.User `json:"user"`
	Token string          `json:"token"`
}

// UpdateProfileRequest содержит данные для обновления профиля
type UpdateProfileRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

// ServiceResponse представляет услугу в API ответе
type ServiceResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ImageURL    string    `json:"image_url"`
	IsActive    bool      `json:"is_active"`
	Density     *float64  `json:"density,omitempty"`
	Thickness   *float64  `json:"thickness,omitempty"`
	Material    *string   `json:"material,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// RequestResponse представляет заявку в API ответе
type RequestResponse struct {
	ID             int        `json:"id"`
	Status         string     `json:"status"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	CreatorID      int        `json:"creator_id"`
	CreatorLogin   string     `json:"creator_login"`
	ModeratorID    *int       `json:"moderator_id,omitempty"`
	ModeratorLogin *string    `json:"moderator_login,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	FormedAt       *time.Time `json:"formed_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	TotalCost      *float64   `json:"total_cost,omitempty"`
	DeliveryDate   *time.Time `json:"delivery_date,omitempty"`
}

// RequestServiceResponse представляет связь заявка-услуга в API ответе
type RequestServiceResponse struct {
	RequestID     int             `json:"request_id"`
	ServiceID     int             `json:"service_id"`
	Quantity      int             `json:"quantity"`
	SortOrder     int             `json:"sort_order"`
	IsMain        bool            `json:"is_main"`
	Comment       string          `json:"comment"`
	ResultFreq    *float64        `json:"result_freq,omitempty"`
	ResultPercent *float64        `json:"result_percent,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	Service       ServiceResponse `json:"service"`
}

// UserResponse представляет пользователя в API ответе
type UserResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	IsActive bool   `json:"is_active"`
}

// ConvertToServiceResponse конвертирует DBService в ServiceResponse
func ConvertToServiceResponse(service repository.DBService) ServiceResponse {
	var createdAt time.Time
	if service.CreatedAt != nil {
		createdAt = *service.CreatedAt
	}

	return ServiceResponse{
		ID:          service.ID,
		Name:        service.Name,
		Description: service.Description,
		ImageURL:    service.ImageURL,
		IsActive:    service.IsActive,
		Density:     service.Density,
		Thickness:   service.Thickness,
		Material:    service.Material,
		CreatedAt:   createdAt,
	}
}

// ConvertToRequestResponse конвертирует RequestWithUsers в RequestResponse
func ConvertToRequestResponse(req RequestWithUsers) RequestResponse {
	return RequestResponse{
		ID:             req.ID,
		Status:         req.Status,
		Title:          req.Title,
		Description:    req.Description,
		CreatorID:      req.CreatorID,
		CreatorLogin:   req.CreatorLogin,
		ModeratorID:    req.ModeratorID,
		ModeratorLogin: req.ModeratorLogin,
		CreatedAt:      req.CreatedAt,
		FormedAt:       req.FormedAt,
		CompletedAt:    req.CompletedAt,
		TotalCost:      req.TotalCost,
		DeliveryDate:   req.DeliveryDate,
	}
}

// ConvertToRequestServiceResponse конвертирует RequestService в RequestServiceResponse
func ConvertToRequestServiceResponse(rs repository.RequestService) RequestServiceResponse {
	return RequestServiceResponse{
		RequestID:     rs.RequestID,
		ServiceID:     rs.ServiceID,
		Quantity:      rs.Quantity,
		SortOrder:     rs.SortOrder,
		IsMain:        rs.IsMain,
		Comment:       rs.Comment,
		ResultFreq:    rs.ResultFreq,
		ResultPercent: rs.ResultPercent,
		CreatedAt:     rs.CreatedAt,
		Service:       ConvertToServiceResponse(rs.Service),
	}
}

// ConvertToUserResponse конвертирует User в UserResponse
func ConvertToUserResponse(user repository.User) UserResponse {
	return UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		FullName: user.FullName,
		IsActive: user.IsActive,
	}
}
