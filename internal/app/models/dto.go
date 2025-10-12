package models

import (
	"time"

	"rip/internal/app/repository"
)

// MaterialFilters содержит фильтры для поиска материалов
type MaterialFilters struct {
	Name         string   `json:"name"`
	Material     string   `json:"material"`
	ThicknessMin *float64 `json:"thickness_min"`
	ThicknessMax *float64 `json:"thickness_max"`
	DensityMin   *float64 `json:"density_min"`
	DensityMax   *float64 `json:"density_max"`
	Page         int      `json:"page"`
	Limit        int      `json:"limit"`
}

// CreateMaterialRequest содержит данные для создания материала
type CreateMaterialRequest struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description"`
	ImageURL    string   `json:"image_url"`
	Density     *float64 `json:"density"`
	Thickness   *float64 `json:"thickness"`
	Material    *string  `json:"material"`
}

// UpdateMaterialRequest содержит данные для обновления материала
type UpdateMaterialRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ImageURL    string   `json:"image_url"`
	Density     *float64 `json:"density"`
	Thickness   *float64 `json:"thickness"`
	Material    *string  `json:"material"`
}

// CalculationFilters содержит фильтры для поиска расчётов
type CalculationFilters struct {
	Status     string     `json:"status"`
	FormedFrom *time.Time `json:"formed_from"`
	FormedTo   *time.Time `json:"formed_to"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
}

// CalculationWithUsers содержит расчёт с информацией о пользователях
type CalculationWithUsers struct {
	repository.Calculation
	CreatorLogin   string  `json:"creator_login"`
	ModeratorLogin *string `json:"moderator_login,omitempty"`
}

// CalculationWithMaterials содержит расчёт с материалами
type CalculationWithMaterials struct {
	repository.Calculation
	CreatorLogin         string                           `json:"creator_login"`
	ModeratorLogin       *string                          `json:"moderator_login,omitempty"`
	MaterialCalculations []repository.MaterialCalculation `json:"material_calculations"`
}

// FormCalculationRequest содержит данные для формирования расчёта
type FormCalculationRequest struct {
	Title                      string   `json:"title" validate:"required"`
	Description                string   `json:"description"`
	OwnFrequency               *float64 `json:"own_frequency" validate:"required"`
	IsolatedInstallationWeight *float64 `json:"isolated_installation_weight" validate:"required"`
}

// CartInfo содержит информацию о корзине
type CartInfo struct {
	CalculationID int `json:"calculation_id"`
	ItemCount     int `json:"item_count"`
}

// UpdateMaterialCalculationRequest содержит данные для обновления связи расчёт-материал
type UpdateMaterialCalculationRequest struct {
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

// MaterialResponse представляет материал в API ответе
type MaterialResponse struct {
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

// CalculationResponse представляет расчёт в API ответе
type CalculationResponse struct {
	ID                         int        `json:"id"`
	Status                     string     `json:"status"`
	Title                      string     `json:"title"`
	Description                string     `json:"description"`
	CreatorID                  int        `json:"creator_id"`
	CreatorLogin               string     `json:"creator_login"`
	ModeratorID                *int       `json:"moderator_id,omitempty"`
	ModeratorLogin             *string    `json:"moderator_login,omitempty"`
	CreatedAt                  time.Time  `json:"created_at"`
	FormedAt                   *time.Time `json:"formed_at,omitempty"`
	CompletedAt                *time.Time `json:"completed_at,omitempty"`
	TotalCost                  *float64   `json:"total_cost,omitempty"`
	DeliveryDate               *time.Time `json:"delivery_date,omitempty"`
	OwnFrequency               *float64   `json:"own_frequency,omitempty"`
	IsolatedInstallationWeight *float64   `json:"isolated_installation_weight,omitempty"`
}

// MaterialCalculationResponse представляет связь расчёт-материал в API ответе
type MaterialCalculationResponse struct {
	CalculationID int              `json:"calculation_id"`
	MaterialID    int              `json:"material_id"`
	Quantity      int              `json:"quantity"`
	SortOrder     int              `json:"sort_order"`
	IsMain        bool             `json:"is_main"`
	Comment       string           `json:"comment"`
	ResultFreq    *float64         `json:"result_freq,omitempty"`
	ResultPercent *float64         `json:"result_percent,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	Material      MaterialResponse `json:"material"`
}

// UserResponse представляет пользователя в API ответе
type UserResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	IsActive bool   `json:"is_active"`
}

// ConvertToMaterialResponse конвертирует DBMaterial в MaterialResponse
func ConvertToMaterialResponse(material repository.DBMaterial) MaterialResponse {
	var createdAt time.Time
	if material.CreatedAt != nil {
		createdAt = *material.CreatedAt
	}

	return MaterialResponse{
		ID:          material.ID,
		Name:        material.Name,
		Description: material.Description,
		ImageURL:    material.ImageURL,
		IsActive:    material.IsActive,
		Density:     material.Density,
		Thickness:   material.Thickness,
		Material:    material.Material,
		CreatedAt:   createdAt,
	}
}

// ConvertToCalculationResponse конвертирует CalculationWithUsers в CalculationResponse
func ConvertToCalculationResponse(calc CalculationWithUsers) CalculationResponse {
	return CalculationResponse{
		ID:                         calc.ID,
		Status:                     calc.Status,
		Title:                      calc.Title,
		Description:                calc.Description,
		CreatorID:                  calc.CreatorID,
		CreatorLogin:               calc.CreatorLogin,
		ModeratorID:                calc.ModeratorID,
		ModeratorLogin:             calc.ModeratorLogin,
		CreatedAt:                  calc.CreatedAt,
		FormedAt:                   calc.FormedAt,
		CompletedAt:                calc.CompletedAt,
		TotalCost:                  calc.TotalCost,
		DeliveryDate:               calc.DeliveryDate,
		OwnFrequency:               calc.OwnFrequency,
		IsolatedInstallationWeight: calc.IsolatedInstallationWeight,
	}
}

// ConvertToMaterialCalculationResponse конвертирует MaterialCalculation в MaterialCalculationResponse
func ConvertToMaterialCalculationResponse(mc repository.MaterialCalculation) MaterialCalculationResponse {
	return MaterialCalculationResponse{
		CalculationID: mc.CalculationID,
		MaterialID:    mc.MaterialID,
		Quantity:      mc.Quantity,
		SortOrder:     mc.SortOrder,
		IsMain:        mc.IsMain,
		Comment:       mc.Comment,
		ResultFreq:    mc.ResultFreq,
		ResultPercent: mc.ResultPercent,
		CreatedAt:     mc.CreatedAt,
		Material:      ConvertToMaterialResponse(mc.Material),
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

// CalculationResult представляет результат вычислений для материала в расчёте
type CalculationResult struct {
	MaterialID    int     `json:"material_id"`
	MaterialName  string  `json:"material_name"`
	Quantity      int     `json:"quantity"`
	ResultFreq    float64 `json:"result_freq"`
	ResultPercent float64 `json:"result_percent"`
	UnitCost      float64 `json:"unit_cost"`
	TotalCost     float64 `json:"total_cost"`
}

// CompleteCalculationResponse представляет ответ при завершении расчёта
type CompleteCalculationResponse struct {
	CalculationID      int                 `json:"calculation_id"`
	Status             string              `json:"status"`
	TotalCost          float64             `json:"total_cost"`
	DeliveryDate       *time.Time          `json:"delivery_date,omitempty"`
	CalculationResults []CalculationResult `json:"calculation_results"`
	Message            string              `json:"message"`
}
