package testdb

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SharedPostgres mantém um único container Postgres para toda a suite de testes.
// Isso é o cenário realista pro CI: sobe 1 container, roda N testes, derruba.
type SharedPostgres struct {
	container testcontainers.Container
	connStr   string
	mu        sync.Mutex
	started   bool

	StartupTime time.Duration
}

func NewSharedPostgres() *SharedPostgres {
	return &SharedPostgres{}
}

// Start sobe o container uma única vez. Chamado no TestMain.
func (s *SharedPostgres) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	ctx := context.Background()
	start := time.Now()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to start postgres: %w", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to get connection string: %w", err)
	}

	s.container = pgContainer
	s.connStr = connStr
	s.started = true
	s.StartupTime = time.Since(start)

	return nil
}

// Stop derruba o container. Chamado no TestMain com defer.
func (s *SharedPostgres) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.container != nil {
		_ = s.container.Terminate(context.Background())
		s.started = false
	}
}

// NewDB cria uma conexão GORM ao container compartilhado e faz AutoMigrate.
// Cada teste recebe sua própria conexão, mas o container é o mesmo.
// Limpa as tabelas antes de cada teste pra garantir isolamento.
func (s *SharedPostgres) NewDB(t *testing.T, entities ...interface{}) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(gormpostgres.Open(s.connStr), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to shared postgres: %v", err)
	}

	if err := db.AutoMigrate(entities...); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}

	// Limpa dados entre testes
	for _, e := range entities {
		db.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(e)
	}

	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})

	return db
}
