package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"rip/internal/app/models"
	"rip/internal/app/repository"

	"gorm.io/gorm"
)

// Service содержит бизнес-логику приложения
type Service struct {
	db                         *gorm.DB
	MaterialService            *MaterialService
	CalculationService         *CalculationService
	MaterialCalculationService *MaterialCalculationService
	UserService                *UserService
}

// NewService создает новый сервис
func NewService(db *gorm.DB) *Service {
	svc := &Service{db: db}
	svc.MaterialService = NewMaterialService(svc)
	svc.CalculationService = NewCalculationService(svc)
	svc.MaterialCalculationService = NewMaterialCalculationService(svc)
	svc.UserService = NewUserService(svc)
	return svc
}

// GetCurrentUserID возвращает ID текущего пользователя (заглушка)
func (s *Service) GetCurrentUserID() int {
	return 1 // Константа согласно ТЗ
}

// GetCurrentModeratorID возвращает ID текущего модератора (заглушка)
func (s *Service) GetCurrentModeratorID() int {
	return 2 // Константа для модератора
}

// MaterialService содержит методы для работы с материалами
type MaterialService struct {
	*Service
}

// NewMaterialService создает новый сервис материалов
func NewMaterialService(svc *Service) *MaterialService {
	return &MaterialService{Service: svc}
}

// GetMaterials возвращает список материалов с фильтрацией
func (s *MaterialService) GetMaterials(ctx context.Context, filters models.MaterialFilters) ([]repository.DBMaterial, int64, error) {
	var materials []repository.DBMaterial
	var total int64

	query := s.db.Model(&repository.DBMaterial{}).Where("is_active = ?", true)

	// Применяем фильтры
	if filters.Name != "" {
		query = query.Where("LOWER(name) LIKE ?", "%"+filters.Name+"%")
	}
	if filters.Material != "" {
		query = query.Where("LOWER(material) LIKE ?", "%"+filters.Material+"%")
	}
	if filters.ThicknessMin != nil {
		query = query.Where("thickness >= ?", *filters.ThicknessMin)
	}
	if filters.ThicknessMax != nil {
		query = query.Where("thickness <= ?", *filters.ThicknessMax)
	}
	if filters.DensityMin != nil {
		query = query.Where("density >= ?", *filters.DensityMin)
	}
	if filters.DensityMax != nil {
		query = query.Where("density <= ?", *filters.DensityMax)
	}

	// Подсчитываем общее количество
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Применяем пагинацию и сортировку
	offset := (filters.Page - 1) * filters.Limit
	if err := query.Order("id").Offset(offset).Limit(filters.Limit).Find(&materials).Error; err != nil {
		return nil, 0, err
	}

	return materials, total, nil
}

// GetMaterial возвращает материал по ID
func (s *MaterialService) GetMaterial(ctx context.Context, id int) (*repository.DBMaterial, error) {
	var material repository.DBMaterial
	if err := s.db.Where("id = ? AND is_active = ?", id, true).First(&material).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMaterialNotFound
		}
		return nil, err
	}
	return &material, nil
}

// CreateMaterial создает новый материал
func (s *MaterialService) CreateMaterial(ctx context.Context, req models.CreateMaterialRequest) (*repository.DBMaterial, error) {
	material := repository.DBMaterial{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
		ImageURL:    req.ImageURL,
		Density:     req.Density,
		Thickness:   req.Thickness,
		Material:    req.Material,
	}

	if err := s.db.Create(&material).Error; err != nil {
		return nil, err
	}

	return &material, nil
}

// UpdateMaterial обновляет материал
func (s *MaterialService) UpdateMaterial(ctx context.Context, id int, req models.UpdateMaterialRequest) (*repository.DBMaterial, error) {
	var material repository.DBMaterial
	if err := s.db.Where("id = ? AND is_active = ?", id, true).First(&material).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMaterialNotFound
		}
		return nil, err
	}

	// Обновляем только переданные поля
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.ImageURL != "" {
		updates["image_url"] = req.ImageURL
	}
	if req.Density != nil {
		updates["density"] = *req.Density
	}
	if req.Thickness != nil {
		updates["thickness"] = *req.Thickness
	}
	if req.Material != nil {
		updates["material"] = *req.Material
	}

	if err := s.db.Model(&material).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &material, nil
}

// DeleteMaterial удаляет материал (логическое удаление)
func (s *MaterialService) DeleteMaterial(ctx context.Context, id int) error {
	var material repository.DBMaterial
	if err := s.db.Where("id = ? AND is_active = ?", id, true).First(&material).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMaterialNotFound
		}
		return err
	}

	// Логическое удаление
	if err := s.db.Model(&material).Update("is_active", false).Error; err != nil {
		return err
	}

	// TODO: Удалить изображение из Minio

	return nil
}

// AddMaterialToCart добавляет материал в корзину (создает расчёт-черновик)
func (s *MaterialService) AddMaterialToCart(ctx context.Context, materialID int) (*repository.Calculation, error) {
	userID := s.GetCurrentUserID()

	// Проверяем существование материала
	var material repository.DBMaterial
	if err := s.db.Where("id = ? AND is_active = ?", materialID, true).First(&material).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMaterialNotFound
		}
		return nil, err
	}

	// Ищем существующий расчёт-черновик пользователя
	var calculation repository.Calculation
	err := s.db.Where("creator_id = ? AND status = ?", userID, "pending").First(&calculation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Создаем новый расчёт-черновик
			calculation = repository.Calculation{
				CreatorID: userID,
				Status:    "pending",
			}
			if err := s.db.Create(&calculation).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Добавляем материал в расчёт
	materialCalculation := repository.MaterialCalculation{
		CalculationID: calculation.ID,
		MaterialID:    materialID,
		Quantity:      1,
	}

	// Используем upsert для обновления количества если материал уже есть
	if err := s.db.Exec(`
		INSERT INTO material_calculation (calculation_id, material_id, quantity, sort_order, is_main, comment, created_at)
		VALUES ($1, $2, 1, 0, false, '', NOW())
		ON CONFLICT (calculation_id, material_id)
		DO UPDATE SET quantity = material_calculation.quantity + 1
	`, materialCalculation.CalculationID, materialCalculation.MaterialID).Error; err != nil {
		return nil, err
	}

	return &calculation, nil
}

// CalculationService содержит методы для работы с расчётами
type CalculationService struct {
	*Service
}

// NewCalculationService создает новый сервис расчётов
func NewCalculationService(svc *Service) *CalculationService {
	return &CalculationService{Service: svc}
}

// GetCartInfo возвращает информацию о корзине текущего пользователя
func (s *CalculationService) GetCartInfo(ctx context.Context) (*models.CartInfo, error) {
	userID := s.GetCurrentUserID()

	var calculation repository.Calculation
	err := s.db.Where("creator_id = ? AND status = ?", userID, "pending").First(&calculation).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &models.CartInfo{CalculationID: 0, ItemCount: 0}, nil
		}
		return nil, err
	}

	var count int64
	if err := s.db.Model(&repository.MaterialCalculation{}).Where("calculation_id = ?", calculation.ID).Count(&count).Error; err != nil {
		return nil, err
	}

	return &models.CartInfo{
		CalculationID: calculation.ID,
		ItemCount:     int(count),
	}, nil
}

// GetCalculations возвращает список расчётов с фильтрацией
func (s *CalculationService) GetCalculations(ctx context.Context, filters models.CalculationFilters) ([]models.CalculationWithUsers, int64, error) {
	var calculations []repository.Calculation
	var total int64

	query := s.db.Model(&repository.Calculation{}).Where("status NOT IN (?)", []string{"deleted", "draft"})

	// Применяем фильтры
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.FormedFrom != nil {
		query = query.Where("formed_at >= ?", *filters.FormedFrom)
	}
	if filters.FormedTo != nil {
		query = query.Where("formed_at <= ?", *filters.FormedTo)
	}

	// Подсчитываем общее количество
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Применяем пагинацию и сортировку
	offset := (filters.Page - 1) * filters.Limit
	if err := query.Order("id DESC").Offset(offset).Limit(filters.Limit).Find(&calculations).Error; err != nil {
		return nil, 0, err
	}

	// Загружаем информацию о пользователях
	var result []models.CalculationWithUsers
	for _, calc := range calculations {
		calculationWithUsers := models.CalculationWithUsers{
			Calculation: calc,
		}

		// Загружаем создателя
		var creator repository.User
		if err := s.db.Where("id = ?", calc.CreatorID).First(&creator).Error; err == nil {
			calculationWithUsers.CreatorLogin = creator.Username
		}

		// Загружаем модератора если есть
		if calc.ModeratorID != nil {
			var moderator repository.User
			if err := s.db.Where("id = ?", *calc.ModeratorID).First(&moderator).Error; err == nil {
				calculationWithUsers.ModeratorLogin = &moderator.Username
			}
		}

		result = append(result, calculationWithUsers)
	}

	return result, total, nil
}

// GetCalculation возвращает расчёт с материалами
func (s *CalculationService) GetCalculation(ctx context.Context, id int) (*models.CalculationWithMaterials, error) {
	var calculation repository.Calculation
	if err := s.db.Where("id = ? AND status NOT IN (?)", id, []string{"deleted", "draft"}).First(&calculation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCalculationNotFound
		}
		return nil, err
	}

	// Загружаем материалы расчёта
	var materialCalculations []repository.MaterialCalculation
	if err := s.db.Preload("Material").Where("calculation_id = ?", id).Order("sort_order, material_id").Find(&materialCalculations).Error; err != nil {
		return nil, err
	}

	// Загружаем информацию о пользователях
	var creator repository.User
	if err := s.db.Where("id = ?", calculation.CreatorID).First(&creator).Error; err == nil {
		// creator login уже загружен
	}

	var moderatorLogin *string
	if calculation.ModeratorID != nil {
		var moderator repository.User
		if err := s.db.Where("id = ?", *calculation.ModeratorID).First(&moderator).Error; err == nil {
			moderatorLogin = &moderator.Username
		}
	}

	return &models.CalculationWithMaterials{
		Calculation:          calculation,
		CreatorLogin:         creator.Username,
		ModeratorLogin:       moderatorLogin,
		MaterialCalculations: materialCalculations,
	}, nil
}

// UpdateCalculation обновляет поля расчёта
func (s *CalculationService) UpdateCalculation(ctx context.Context, id int, req models.UpdateCalculationRequest) error {
	var calculation repository.Calculation
	if err := s.db.Where("id = ? AND status NOT IN (?)", id, []string{"deleted", "draft"}).First(&calculation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCalculationNotFound
		}
		return err
	}

	// Обновляем только переданные поля
	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if len(updates) > 0 {
		if err := s.db.Model(&calculation).Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}

// FormCalculation формирует расчёт (переводит из черновика в сформированный)
func (s *CalculationService) FormCalculation(ctx context.Context, id int) error {
	userID := s.GetCurrentUserID()

	var calculation repository.Calculation
	if err := s.db.Where("id = ? AND creator_id = ? AND status = ?", id, userID, "pending").First(&calculation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCalculationNotFound
		}
		return err
	}

	// Проверяем обязательные поля
	if calculation.Title == "" {
		return ErrCalculationMissingRequiredFields
	}

	// Обновляем статус и дату формирования
	now := time.Now()
	if err := s.db.Model(&calculation).Updates(map[string]interface{}{
		"status":    "completed",
		"formed_at": &now,
	}).Error; err != nil {
		return err
	}

	return nil
}

// CompleteCalculation завершает или отклоняет расчёт модератором
func (s *CalculationService) CompleteCalculation(ctx context.Context, id int, action string) (*models.CompleteCalculationResponse, error) {
	moderatorID := s.GetCurrentModeratorID()

	var calculation repository.Calculation
	if err := s.db.Where("id = ? AND status = ?", id, "completed").First(&calculation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCalculationNotFound
		}
		return nil, err
	}

	var newStatus string
	switch action {
	case "complete":
		newStatus = "completed"
	case "reject":
		newStatus = "rejected"
	default:
		return nil, ErrInvalidAction
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":       newStatus,
		"completed_at": &now,
		"moderator_id": moderatorID,
	}

	var response *models.CompleteCalculationResponse

	// При завершении выполняем расчеты
	if action == "complete" {
		// Загружаем материалы расчёта для расчета
		var materialCalculations []repository.MaterialCalculation
		if err := s.db.Preload("Material").Where("calculation_id = ?", id).Find(&materialCalculations).Error; err != nil {
			return nil, err
		}

		// Рассчитываем стоимость заказа
		totalCost := s.calculateOrderCost(materialCalculations)
		updates["total_cost"] = totalCost

		// Рассчитываем дату доставки (в течение месяца)
		deliveryDate := now.AddDate(0, 0, 30) // 30 дней
		updates["delivery_date"] = &deliveryDate

		// Создаем результаты вычислений
		var calculationResults []models.CalculationResult

		// Обновляем результаты расчета в м-м таблице и собираем результаты
		for _, mc := range materialCalculations {
			var resultFreq, resultPercent float64
			var unitCost float64

			if mc.Material.Density != nil && mc.Material.Thickness != nil {
				// Простая формула расчета (пример)
				resultFreq = math.Sqrt(1000.0/float64(mc.Quantity)) / (2 * math.Pi)
				resultPercent = 100.0 * (1 - (resultFreq / (50.0 + resultFreq))) // 50 Hz - частота вибрации

				if err := s.db.Model(&mc).Updates(map[string]interface{}{
					"result_freq":    resultFreq,
					"result_percent": resultPercent,
				}).Error; err != nil {
					return nil, err
				}
			}

			// Рассчитываем стоимость единицы товара
			unitCost = s.calculateUnitCost(mc.Material)
			totalItemCost := unitCost * float64(mc.Quantity)

			calculationResults = append(calculationResults, models.CalculationResult{
				MaterialID:    mc.MaterialID,
				MaterialName:  mc.Material.Name,
				Quantity:      mc.Quantity,
				ResultFreq:    resultFreq,
				ResultPercent: resultPercent,
				UnitCost:      unitCost,
				TotalCost:     totalItemCost,
			})
		}

		response = &models.CompleteCalculationResponse{
			CalculationID:      id,
			Status:             newStatus,
			TotalCost:          totalCost,
			DeliveryDate:       &deliveryDate,
			CalculationResults: calculationResults,
			Message:            "Расчёт завершён",
		}
	} else {
		// Для отклонения расчёта
		response = &models.CompleteCalculationResponse{
			CalculationID:      id,
			Status:             newStatus,
			TotalCost:          0,
			DeliveryDate:       nil,
			CalculationResults: []models.CalculationResult{},
			Message:            "Расчёт отклонён",
		}
	}

	if err := s.db.Model(&calculation).Updates(updates).Error; err != nil {
		return nil, err
	}

	return response, nil
}

// DeleteCalculation удаляет расчёт (логическое удаление)
func (s *CalculationService) DeleteCalculation(ctx context.Context, id int) error {
	userID := s.GetCurrentUserID()

	var calculation repository.Calculation
	if err := s.db.Where("id = ? AND creator_id = ? AND status IN (?)", id, userID, []string{"pending", "completed", "rejected"}).First(&calculation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCalculationNotFound
		}
		return err
	}

	// Логическое удаление
	if err := s.db.Model(&calculation).Update("status", "rejected").Error; err != nil {
		return err
	}

	return nil
}

// calculateOrderCost рассчитывает стоимость заказа
func (s *CalculationService) calculateOrderCost(materialCalculations []repository.MaterialCalculation) float64 {
	totalCost := 0.0
	for _, mc := range materialCalculations {
		unitCost := s.calculateUnitCost(mc.Material)
		totalCost += unitCost * float64(mc.Quantity)
	}
	return totalCost
}

// calculateUnitCost рассчитывает стоимость единицы товара
func (s *CalculationService) calculateUnitCost(material repository.DBMaterial) float64 {
	// Простая формула расчета стоимости (пример)
	basePrice := 100.0 // Базовая цена
	if material.Density != nil {
		basePrice += *material.Density * 0.1
	}
	if material.Thickness != nil {
		basePrice += *material.Thickness * 2.0
	}
	return basePrice
}

// MaterialCalculationService содержит методы для работы со связью расчёт-материал
type MaterialCalculationService struct {
	*Service
}

// NewMaterialCalculationService создает новый сервис связи расчёт-материал
func NewMaterialCalculationService(svc *Service) *MaterialCalculationService {
	return &MaterialCalculationService{Service: svc}
}

// DeleteMaterialCalculation удаляет материал из расчёта
func (s *MaterialCalculationService) DeleteMaterialCalculation(ctx context.Context, calculationID, materialID int) error {
	userID := s.GetCurrentUserID()

	// Проверяем права доступа
	var calculation repository.Calculation
	if err := s.db.Where("id = ? AND creator_id = ?", calculationID, userID).First(&calculation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCalculationNotFound
		}
		return err
	}

	// Удаляем связь
	if err := s.db.Where("calculation_id = ? AND material_id = ?", calculationID, materialID).Delete(&repository.MaterialCalculation{}).Error; err != nil {
		return err
	}

	return nil
}

// UpdateMaterialCalculation обновляет связь расчёт-материал
func (s *MaterialCalculationService) UpdateMaterialCalculation(ctx context.Context, calculationID, materialID int, req models.UpdateMaterialCalculationRequest) error {
	userID := s.GetCurrentUserID()

	// Проверяем права доступа
	var calculation repository.Calculation
	if err := s.db.Where("id = ? AND creator_id = ?", calculationID, userID).First(&calculation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCalculationNotFound
		}
		return err
	}

	// Обновляем связь
	updates := make(map[string]interface{})
	if req.Quantity != nil {
		updates["quantity"] = *req.Quantity
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.IsMain != nil {
		updates["is_main"] = *req.IsMain
	}
	if req.Comment != "" {
		updates["comment"] = req.Comment
	}

	if len(updates) > 0 {
		if err := s.db.Model(&repository.MaterialCalculation{}).Where("calculation_id = ? AND material_id = ?", calculationID, materialID).Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}

// UserService содержит методы для работы с пользователями
type UserService struct {
	*Service
}

// NewUserService создает новый сервис пользователей
func NewUserService(svc *Service) *UserService {
	return &UserService{Service: svc}
}

// Register регистрирует нового пользователя
func (s *UserService) Register(ctx context.Context, req models.RegisterRequest) (*repository.User, error) {
	// Проверяем уникальность username и email
	var existingUser repository.User
	if err := s.db.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error; err == nil {
		return nil, ErrUserAlreadyExists
	}

	// Хешируем пароль (в реальном приложении используйте bcrypt)
	hashedPassword := fmt.Sprintf("hashed_%s", req.Password)

	user := repository.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
		FullName: req.FullName,
		IsActive: true,
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// Login аутентифицирует пользователя
func (s *UserService) Login(ctx context.Context, req models.LoginRequest) (*models.LoginResponse, error) {
	var user repository.User
	if err := s.db.Where("username = ? AND is_active = ?", req.Username, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Проверяем пароль (в реальном приложении используйте bcrypt)
	if user.Password != fmt.Sprintf("hashed_%s", req.Password) {
		return nil, ErrInvalidCredentials
	}

	// Генерируем токен (в реальном приложении используйте JWT)
	token := fmt.Sprintf("token_%d_%d", user.ID, time.Now().Unix())

	return &models.LoginResponse{
		User:  user,
		Token: token,
	}, nil
}

// GetProfile возвращает профиль пользователя
func (s *UserService) GetProfile(ctx context.Context, userID int) (*repository.User, error) {
	var user repository.User
	if err := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// UpdateProfile обновляет профиль пользователя
func (s *UserService) UpdateProfile(ctx context.Context, userID int, req models.UpdateProfileRequest) (*repository.User, error) {
	var user repository.User
	if err := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Обновляем только переданные поля
	updates := make(map[string]interface{})
	if req.FullName != "" {
		updates["full_name"] = req.FullName
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}

	if len(updates) > 0 {
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return &user, nil
}

// Ошибки сервиса
var (
	ErrMaterialNotFound                 = errors.New("материал не найден")
	ErrCalculationNotFound              = errors.New("расчёт не найден")
	ErrUserNotFound                     = errors.New("пользователь не найден")
	ErrUserAlreadyExists                = errors.New("пользователь уже существует")
	ErrInvalidCredentials               = errors.New("неверные учетные данные")
	ErrCalculationMissingRequiredFields = errors.New("отсутствуют обязательные поля расчёта")
	ErrInvalidAction                    = errors.New("неверное действие")
)
