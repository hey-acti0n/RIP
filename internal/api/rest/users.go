package rest

import (
	"encoding/json"
	"net/http"

	"rip/internal/app/models"
	"rip/internal/app/service"
)

// Register регистрирует нового пользователя
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
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// TODO: Извлечь userID из токена авторизации
	// Пока используем заглушку
	userID := h.service.GetCurrentUserID()

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
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// TODO: Извлечь userID из токена авторизации
	// Пока используем заглушку
	userID := h.service.GetCurrentUserID()

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
func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// TODO: Инвалидировать токен
	// Пока возвращаем успешный ответ
	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Выход выполнен успешно"})
}
