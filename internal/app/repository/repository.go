package repository

import "RIP/internal/app"

type Store interface {
}

// For now we expose the concrete store from app. In a real app, define methods here.
func NewInMemory() *app.Store {
	return app.NewStore()
}
