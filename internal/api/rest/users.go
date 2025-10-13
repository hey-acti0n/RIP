package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"rip/internal/app/models"
	"rip/internal/app/service"
)

// Register регистрирует нового пользователя
// @Summary Регистрация пользователя
// @Description Создает нового пользователя в системе
// @Tags users
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Данные для регистрации"
// @Success 201 {object} models.UserResponse "Пользователь успешно создан"
// @Failure 400 {object} map[string]string "Неверные данные"
// @Failure 409 {object} map[string]string "Пользователь уже существует"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /users/register [post]
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}

	// Валидация
	if req.Username == "" {
		h.writeError(w, http.StatusBadRequest, "Имя пользователя обязательно")
		return
	}
	if req.Email == "" {
		h.writeError(w, http.StatusBadRequest, "Email обязателен")
		return
	}
	if req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "Пароль обязателен")
		return
	}
	if len(req.Password) < 6 {
		h.writeError(w, http.StatusBadRequest, "Пароль должен содержать минимум 6 символов")
		return
	}
	if req.FullName == "" {
		h.writeError(w, http.StatusBadRequest, "Полное имя обязательно")
		return
	}

	user, err := h.service.UserService.Register(r.Context(), req)
	if err != nil {
		if err == service.ErrUserAlreadyExists {
			h.writeError(w, http.StatusConflict, "Пользователь с таким именем или email уже существует")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка регистрации пользователя")
		return
	}

	response := models.ConvertToUserResponse(*user)
	h.writeJSON(w, http.StatusCreated, response)
}

// Login аутентифицирует пользователя
// @Summary Аутентификация пользователя
// @Description Аутентифицирует пользователя и возвращает JWT токен
// @Tags users
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Данные для входа"
// @Success 200 {object} map[string]interface{} "Успешная аутентификация"
// @Failure 400 {object} map[string]string "Неверные данные"
// @Failure 401 {object} map[string]string "Неверные учетные данные"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /users/login [post]
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}

	// Валидация
	if req.Username == "" {
		h.writeError(w, http.StatusBadRequest, "Имя пользователя обязательно")
		return
	}
	if req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "Пароль обязателен")
		return
	}

	loginResponse, err := h.service.UserService.Login(r.Context(), req)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			h.writeError(w, http.StatusUnauthorized, "Неверные учетные данные")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка аутентификации")
		return
	}

	response := struct {
		User  models.UserResponse `json:"user"`
		Token string              `json:"token"`
	}{
		User:  models.ConvertToUserResponse(loginResponse.User),
		Token: loginResponse.Token,
	}

	h.writeJSON(w, http.StatusOK, response)
}

// GetProfile возвращает профиль пользователя
// @Summary Получить профиль пользователя
// @Description Возвращает информацию о профиле текущего пользователя
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.UserResponse "Профиль пользователя"
// @Failure 401 {object} map[string]string "Пользователь не аутентифицирован"
// @Failure 404 {object} map[string]string "Пользователь не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /users/profile [get]
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Пользователь не аутентифицирован")
		return
	}

	user, err := h.service.UserService.GetProfile(r.Context(), userID)
	if err != nil {
		if err == service.ErrUserNotFound {
			h.writeError(w, http.StatusNotFound, "Пользователь не найден")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка получения профиля")
		return
	}

	response := models.ConvertToUserResponse(*user)
	h.writeJSON(w, http.StatusOK, response)
}

// UpdateProfile обновляет профиль пользователя
// @Summary Обновить профиль пользователя
// @Description Обновляет информацию профиля текущего пользователя
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UpdateProfileRequest true "Данные для обновления"
// @Success 200 {object} models.UserResponse "Профиль обновлен"
// @Failure 400 {object} map[string]string "Неверные данные"
// @Failure 401 {object} map[string]string "Пользователь не аутентифицирован"
// @Failure 404 {object} map[string]string "Пользователь не найден"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /users/profile [put]
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(int)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Пользователь не аутентифицирован")
		return
	}

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Неверный формат данных")
		return
	}

	user, err := h.service.UserService.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		if err == service.ErrUserNotFound {
			h.writeError(w, http.StatusNotFound, "Пользователь не найден")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "Ошибка обновления профиля")
		return
	}

	response := models.ConvertToUserResponse(*user)
	h.writeJSON(w, http.StatusOK, response)
}

// Logout деавторизует пользователя
// @Summary Выход из системы
// @Description Деавторизует пользователя, добавляя токен в черный список
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "Выход выполнен успешно"
// @Failure 401 {object} map[string]string "Пользователь не аутентифицирован"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /users/logout [post]
func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token, ok := r.Context().Value("token").(string)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "Пользователь не аутентифицирован")
		return
	}

	// Добавляем токен в черный список
	if h.service.RedisClient != nil {
		err := h.service.RedisClient.AddToBlacklist(r.Context(), token, 24*time.Hour)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "Ошибка при выходе из системы")
			return
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Выход выполнен успешно"})
}
