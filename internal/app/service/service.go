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
	db                    *gorm.DB
	ServiceService        *ServiceService
	RequestService        *RequestService
	RequestServiceService *RequestServiceService
	UserService           *UserService
}

// NewService создает новый сервис
func NewService(db *gorm.DB) *Service {
	svc := &Service{db: db}
	svc.ServiceService = NewServiceService(svc)
	svc.RequestService = NewRequestService(svc)
	svc.RequestServiceService = NewRequestServiceService(svc)
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

// ServiceService содержит методы для работы с услугами
type ServiceService struct {
	*Service
}

// NewServiceService создает новый сервис услуг
func NewServiceService(svc *Service) *ServiceService {
	return &ServiceService{Service: svc}
}

// GetServices возвращает список услуг с фильтрацией
func (s *ServiceService) GetServices(ctx context.Context, filters models.ServiceFilters) ([]repository.DBService, int64, error) {
	var services []repository.DBService
	var total int64

	query := s.db.Model(&repository.DBService{}).Where("is_active = ?", true)

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
	if err := query.Order("id").Offset(offset).Limit(filters.Limit).Find(&services).Error; err != nil {
		return nil, 0, err
	}

	return services, total, nil
}

// GetService возвращает услугу по ID
func (s *ServiceService) GetService(ctx context.Context, id int) (*repository.DBService, error) {
	var service repository.DBService
	if err := s.db.Where("id = ? AND is_active = ?", id, true).First(&service).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServiceNotFound
		}
		return nil, err
	}
	return &service, nil
}

// CreateService создает новую услугу
func (s *ServiceService) CreateService(ctx context.Context, req models.CreateServiceRequest) (*repository.DBService, error) {
	service := repository.DBService{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    true,
		ImageURL:    req.ImageURL,
		Density:     req.Density,
		Thickness:   req.Thickness,
		Material:    req.Material,
	}

	if err := s.db.Create(&service).Error; err != nil {
		return nil, err
	}

	return &service, nil
}

// UpdateService обновляет услугу
func (s *ServiceService) UpdateService(ctx context.Context, id int, req models.UpdateServiceRequest) (*repository.DBService, error) {
	var service repository.DBService
	if err := s.db.Where("id = ? AND is_active = ?", id, true).First(&service).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServiceNotFound
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

	if err := s.db.Model(&service).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &service, nil
}

// DeleteService удаляет услугу (логическое удаление)
func (s *ServiceService) DeleteService(ctx context.Context, id int) error {
	var service repository.DBService
	if err := s.db.Where("id = ? AND is_active = ?", id, true).First(&service).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrServiceNotFound
		}
		return err
	}

	// Логическое удаление
	if err := s.db.Model(&service).Update("is_active", false).Error; err != nil {
		return err
	}

	// TODO: Удалить изображение из Minio

	return nil
}

// AddServiceToCart добавляет услугу в корзину (создает заявку-черновик)
func (s *ServiceService) AddServiceToCart(ctx context.Context, serviceID int) (*repository.Request, error) {
	userID := s.GetCurrentUserID()

	// Проверяем существование услуги
	var service repository.DBService
	if err := s.db.Where("id = ? AND is_active = ?", serviceID, true).First(&service).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrServiceNotFound
		}
		return nil, err
	}

	// Ищем существующую заявку-черновик пользователя
	var request repository.Request
	err := s.db.Where("creator_id = ? AND status = ?", userID, "pending").First(&request).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Создаем новую заявку-черновик
			request = repository.Request{
				CreatorID: userID,
				Status:    "pending",
			}
			if err := s.db.Create(&request).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// Добавляем услугу в заявку
	requestService := repository.RequestService{
		RequestID: request.ID,
		ServiceID: serviceID,
		Quantity:  1,
	}

	// Используем upsert для обновления количества если услуга уже есть
	if err := s.db.Exec(`
		INSERT INTO request_services (request_id, service_id, quantity, sort_order, is_main, comment, created_at)
		VALUES ($1, $2, 1, 0, false, '', NOW())
		ON CONFLICT (request_id, service_id)
		DO UPDATE SET quantity = request_services.quantity + 1
	`, requestService.RequestID, requestService.ServiceID).Error; err != nil {
		return nil, err
	}

	return &request, nil
}

// RequestService содержит методы для работы с заявками
type RequestService struct {
	*Service
}

// NewRequestService создает новый сервис заявок
func NewRequestService(svc *Service) *RequestService {
	return &RequestService{Service: svc}
}

// GetCartInfo возвращает информацию о корзине текущего пользователя
func (s *RequestService) GetCartInfo(ctx context.Context) (*models.CartInfo, error) {
	userID := s.GetCurrentUserID()

	var request repository.Request
	err := s.db.Where("creator_id = ? AND status = ?", userID, "pending").First(&request).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &models.CartInfo{RequestID: 0, ItemCount: 0}, nil
		}
		return nil, err
	}

	var count int64
	if err := s.db.Model(&repository.RequestService{}).Where("request_id = ?", request.ID).Count(&count).Error; err != nil {
		return nil, err
	}

	return &models.CartInfo{
		RequestID: request.ID,
		ItemCount: int(count),
	}, nil
}

// GetRequests возвращает список заявок с фильтрацией
func (s *RequestService) GetRequests(ctx context.Context, filters models.RequestFilters) ([]models.RequestWithUsers, int64, error) {
	var requests []repository.Request
	var total int64

	query := s.db.Model(&repository.Request{}).Where("status NOT IN (?)", []string{"deleted", "draft"})

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
	if err := query.Order("id DESC").Offset(offset).Limit(filters.Limit).Find(&requests).Error; err != nil {
		return nil, 0, err
	}

	// Загружаем информацию о пользователях
	var result []models.RequestWithUsers
	for _, req := range requests {
		requestWithUsers := models.RequestWithUsers{
			Request: req,
		}

		// Загружаем создателя
		var creator repository.User
		if err := s.db.Where("id = ?", req.CreatorID).First(&creator).Error; err == nil {
			requestWithUsers.CreatorLogin = creator.Username
		}

		// Загружаем модератора если есть
		if req.ModeratorID != nil {
			var moderator repository.User
			if err := s.db.Where("id = ?", *req.ModeratorID).First(&moderator).Error; err == nil {
				requestWithUsers.ModeratorLogin = &moderator.Username
			}
		}

		result = append(result, requestWithUsers)
	}

	return result, total, nil
}

// GetRequest возвращает заявку с услугами
func (s *RequestService) GetRequest(ctx context.Context, id int) (*models.RequestWithServices, error) {
	var request repository.Request
	if err := s.db.Where("id = ? AND status NOT IN (?)", id, []string{"deleted", "draft"}).First(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}

	// Загружаем услуги заявки
	var requestServices []repository.RequestService
	if err := s.db.Preload("Service").Where("request_id = ?", id).Order("sort_order, service_id").Find(&requestServices).Error; err != nil {
		return nil, err
	}

	// Загружаем информацию о пользователях
	var creator repository.User
	if err := s.db.Where("id = ?", request.CreatorID).First(&creator).Error; err == nil {
		// creator login уже загружен
	}

	var moderatorLogin *string
	if request.ModeratorID != nil {
		var moderator repository.User
		if err := s.db.Where("id = ?", *request.ModeratorID).First(&moderator).Error; err == nil {
			moderatorLogin = &moderator.Username
		}
	}

	return &models.RequestWithServices{
		Request:         request,
		CreatorLogin:    creator.Username,
		ModeratorLogin:  moderatorLogin,
		RequestServices: requestServices,
	}, nil
}

// UpdateRequest обновляет поля заявки
func (s *RequestService) UpdateRequest(ctx context.Context, id int, req models.UpdateRequestRequest) error {
	var request repository.Request
	if err := s.db.Where("id = ? AND status NOT IN (?)", id, []string{"deleted", "draft"}).First(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRequestNotFound
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
		if err := s.db.Model(&request).Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}

// FormRequest формирует заявку (переводит из черновика в сформированную)
func (s *RequestService) FormRequest(ctx context.Context, id int) error {
	userID := s.GetCurrentUserID()

	var request repository.Request
	if err := s.db.Where("id = ? AND creator_id = ? AND status = ?", id, userID, "draft").First(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRequestNotFound
		}
		return err
	}

	// Проверяем обязательные поля
	if request.Title == "" {
		return ErrRequestMissingRequiredFields
	}

	// Обновляем статус и дату формирования
	now := time.Now()
	if err := s.db.Model(&request).Updates(map[string]interface{}{
		"status":    "formed",
		"formed_at": &now,
	}).Error; err != nil {
		return err
	}

	return nil
}

// CompleteRequest завершает или отклоняет заявку модератором
func (s *RequestService) CompleteRequest(ctx context.Context, id int, action string) error {
	moderatorID := s.GetCurrentModeratorID()

	var request repository.Request
	if err := s.db.Where("id = ? AND status = ?", id, "formed").First(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRequestNotFound
		}
		return err
	}

	var newStatus string
	switch action {
	case "complete":
		newStatus = "completed"
	case "reject":
		newStatus = "rejected"
	default:
		return ErrInvalidAction
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":       newStatus,
		"completed_at": &now,
		"moderator_id": moderatorID,
	}

	// При завершении выполняем расчеты
	if action == "complete" {
		// Загружаем услуги заявки для расчета
		var requestServices []repository.RequestService
		if err := s.db.Preload("Service").Where("request_id = ?", id).Find(&requestServices).Error; err != nil {
			return err
		}

		// Рассчитываем стоимость заказа
		totalCost := s.calculateOrderCost(requestServices)
		updates["total_cost"] = totalCost

		// Рассчитываем дату доставки (в течение месяца)
		deliveryDate := now.AddDate(0, 0, 30) // 30 дней
		updates["delivery_date"] = &deliveryDate

		// Обновляем результаты расчета в м-м таблице
		for _, rs := range requestServices {
			if rs.Service.Density != nil && rs.Service.Thickness != nil {
				// Простая формула расчета (пример)
				resultFreq := math.Sqrt(1000.0/float64(rs.Quantity)) / (2 * math.Pi)
				resultPercent := 100.0 * (1 - (resultFreq / (50.0 + resultFreq))) // 50 Hz - частота вибрации

				if err := s.db.Model(&rs).Updates(map[string]interface{}{
					"result_freq":    resultFreq,
					"result_percent": resultPercent,
				}).Error; err != nil {
					return err
				}
			}
		}
	}

	if err := s.db.Model(&request).Updates(updates).Error; err != nil {
		return err
	}

	return nil
}

// DeleteRequest удаляет заявку (логическое удаление)
func (s *RequestService) DeleteRequest(ctx context.Context, id int) error {
	userID := s.GetCurrentUserID()

	var request repository.Request
	if err := s.db.Where("id = ? AND creator_id = ? AND status = ?", id, userID, "formed").First(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRequestNotFound
		}
		return err
	}

	// Логическое удаление
	if err := s.db.Model(&request).Update("status", "deleted").Error; err != nil {
		return err
	}

	return nil
}

// calculateOrderCost рассчитывает стоимость заказа
func (s *RequestService) calculateOrderCost(requestServices []repository.RequestService) float64 {
	totalCost := 0.0
	for _, rs := range requestServices {
		// Простая формула расчета стоимости (пример)
		basePrice := 100.0 // Базовая цена
		if rs.Service.Density != nil {
			basePrice += *rs.Service.Density * 0.1
		}
		if rs.Service.Thickness != nil {
			basePrice += *rs.Service.Thickness * 2.0
		}
		totalCost += basePrice * float64(rs.Quantity)
	}
	return totalCost
}

// RequestServiceService содержит методы для работы со связью заявка-услуга
type RequestServiceService struct {
	*Service
}

// NewRequestServiceService создает новый сервис связи заявка-услуга
func NewRequestServiceService(svc *Service) *RequestServiceService {
	return &RequestServiceService{Service: svc}
}

// DeleteRequestService удаляет услугу из заявки
func (s *RequestServiceService) DeleteRequestService(ctx context.Context, requestID, serviceID int) error {
	userID := s.GetCurrentUserID()

	// Проверяем права доступа
	var request repository.Request
	if err := s.db.Where("id = ? AND creator_id = ?", requestID, userID).First(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRequestNotFound
		}
		return err
	}

	// Удаляем связь
	if err := s.db.Where("request_id = ? AND service_id = ?", requestID, serviceID).Delete(&repository.RequestService{}).Error; err != nil {
		return err
	}

	return nil
}

// UpdateRequestService обновляет связь заявка-услуга
func (s *RequestServiceService) UpdateRequestService(ctx context.Context, requestID, serviceID int, req models.UpdateRequestServiceRequest) error {
	userID := s.GetCurrentUserID()

	// Проверяем права доступа
	var request repository.Request
	if err := s.db.Where("id = ? AND creator_id = ?", requestID, userID).First(&request).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRequestNotFound
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
		if err := s.db.Model(&repository.RequestService{}).Where("request_id = ? AND service_id = ?", requestID, serviceID).Updates(updates).Error; err != nil {
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
	ErrServiceNotFound              = errors.New("услуга не найдена")
	ErrRequestNotFound              = errors.New("заявка не найдена")
	ErrUserNotFound                 = errors.New("пользователь не найден")
	ErrUserAlreadyExists            = errors.New("пользователь уже существует")
	ErrInvalidCredentials           = errors.New("неверные учетные данные")
	ErrRequestMissingRequiredFields = errors.New("отсутствуют обязательные поля заявки")
	ErrInvalidAction                = errors.New("неверное действие")
)
