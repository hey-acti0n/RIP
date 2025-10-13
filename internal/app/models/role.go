package models

// Role представляет роль пользователя в системе
type Role int

const (
	// UserRole - обычный пользователь
	UserRole Role = iota
	// ModeratorRole - модератор
	ModeratorRole
	// AdminRole - администратор
	AdminRole
)

// String возвращает строковое представление роли
func (r Role) String() string {
	switch r {
	case UserRole:
		return "user"
	case ModeratorRole:
		return "moderator"
	case AdminRole:
		return "admin"
	default:
		return "unknown"
	}
}

// HasPermission проверяет, имеет ли роль необходимые права
func (r Role) HasPermission(requiredRole Role) bool {
	return r >= requiredRole
}
