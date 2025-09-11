package grpc

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type ConnectionPool struct {
	connections []*grpc.ClientConn
	current     int32
	mu          sync.RWMutex
	endpoint    string
	size        int
}

func NewConnectionPool(endpoint string, size int) (*ConnectionPool, error) {
	if size <= 0 {
		size = 5
	}

	pool := &ConnectionPool{
		connections: make([]*grpc.ClientConn, size),
		endpoint:    endpoint,
		size:        size,
	}

	for i := 0; i < size; i++ {
		conn, err := createConnection(endpoint)
		if err != nil {
			for j := 0; j < i; j++ {
				_ = pool.connections[j].Close()
			}
			return nil, fmt.Errorf("failed to create connection %d: %w", i, err)
		}
		pool.connections[i] = conn
	}

	return pool, nil
}

func createConnection(endpoint string) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(10 * 1024 * 1024),
			grpc.MaxCallSendMsgSize(10 * 1024 * 1024),
		),
	}

	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return conn, nil
}

func (p *ConnectionPool) GetConnection() *grpc.ClientConn {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.connections) == 0 {
		return nil
	}

	next := atomic.AddInt32(&p.current, 1)
	idx := int(next-1) % len(p.connections)

	return p.connections[idx]
}

func (p *ConnectionPool) RefreshConnection(idx int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if idx < 0 || idx >= len(p.connections) {
		return fmt.Errorf("invalid connection index: %d", idx)
	}

	if p.connections[idx] != nil {
		_ = p.connections[idx].Close()
	}

	conn, err := createConnection(p.endpoint)
	if err != nil {
		return fmt.Errorf("failed to refresh connection: %w", err)
	}

	p.connections[idx] = conn
	return nil
}

func (p *ConnectionPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections)
}

func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for i, conn := range p.connections {
		if conn != nil {
			if err := conn.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close connection %d: %w", i, err))
			}
		}
	}

	p.connections = nil

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

type PoolStats struct {
	Endpoint        string
	Size            int
	ActiveConnections int
	CurrentIndex    int32
}

func (p *ConnectionPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	active := 0
	for _, conn := range p.connections {
		if conn != nil && conn.GetState().String() == "READY" {
			active++
		}
	}

	return PoolStats{
		Endpoint:          p.endpoint,
		Size:              len(p.connections),
		ActiveConnections: active,
		CurrentIndex:      atomic.LoadInt32(&p.current),
	}
}