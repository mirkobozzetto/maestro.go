package grpc

import (
	"fmt"
	"sync"
	"time"

	"github.com/maestro/maestro.go/internal/workflow"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
)

type ServiceRegistry struct {
	mu              sync.RWMutex
	services        map[string]*ServiceEntry
	connectionPools map[string]*ConnectionPool
	circuitBreakers map[string]*gobreaker.CircuitBreaker
}

type ServiceEntry struct {
	Config          *workflow.Service
	Healthy         bool
	LastHealthCheck time.Time
	Connection      *grpc.ClientConn
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services:        make(map[string]*ServiceEntry),
		connectionPools: make(map[string]*ConnectionPool),
		circuitBreakers: make(map[string]*gobreaker.CircuitBreaker),
	}
}

func (r *ServiceRegistry) RegisterService(name string, config *workflow.Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[name]; exists {
		return fmt.Errorf("service %s already registered", name)
	}

	entry := &ServiceEntry{
		Config:          config,
		Healthy:         true,
		LastHealthCheck: time.Now(),
	}

	if config.Type == "grpc" {
		pool, err := NewConnectionPool(config.Endpoint, 5)
		if err != nil {
			return fmt.Errorf("failed to create connection pool: %w", err)
		}
		r.connectionPools[name] = pool
	}

	cbSettings := gobreaker.Settings{
		Name:        fmt.Sprintf("%s_circuit_breaker", name),
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= 0.6
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			if to == gobreaker.StateOpen {
				entry.Healthy = false
			} else if to == gobreaker.StateClosed {
				entry.Healthy = true
			}
		},
	}

	r.circuitBreakers[name] = gobreaker.NewCircuitBreaker(cbSettings)
	r.services[name] = entry

	return nil
}

func (r *ServiceRegistry) GetService(name string) (*ServiceEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return entry, nil
}

func (r *ServiceRegistry) GetConnection(serviceName string) (*grpc.ClientConn, error) {
	r.mu.RLock()
	pool, exists := r.connectionPools[serviceName]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no connection pool for service %s", serviceName)
	}

	return pool.GetConnection(), nil
}

func (r *ServiceRegistry) GetCircuitBreaker(serviceName string) (*gobreaker.CircuitBreaker, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cb, exists := r.circuitBreakers[serviceName]
	if !exists {
		return nil, fmt.Errorf("no circuit breaker for service %s", serviceName)
	}

	return cb, nil
}

func (r *ServiceRegistry) UpdateHealth(name string, healthy bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, exists := r.services[name]; exists {
		entry.Healthy = healthy
		entry.LastHealthCheck = time.Now()
	}
}

func (r *ServiceRegistry) IsHealthy(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if entry, exists := r.services[name]; exists {
		return entry.Healthy
	}

	return false
}

func (r *ServiceRegistry) ListServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.services))
	for name := range r.services {
		names = append(names, name)
	}

	return names
}

func (r *ServiceRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for name, pool := range r.connectionPools {
		if err := pool.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close pool for %s: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing registry: %v", errs)
	}

	return nil
}