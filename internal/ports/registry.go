package ports

import "github.com/maestro/maestro.go/internal/domain"

type ServiceRegistry interface {
	RegisterService(name string, config *domain.Service) error
	GetService(name string) (*domain.Service, error)
	IsHealthy(name string) bool
	UpdateHealth(name string, healthy bool)
}