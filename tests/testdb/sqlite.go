package testdb

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewSQLiteDB é o helper atual usado nos projetos.
// Cria um banco SQLite in-memory para testes.
//
// Problemas conhecidos:
// - Não suporta JSONB, UUID nativo, INTERVAL, ARRAY
// - Comportamento diferente em COALESCE, type casting, NOW()
// - Migrations com SQL raw pra Postgres quebram aqui
// - Falsa segurança: testes passam mas podem falhar em prod
func NewSQLiteDB(t *testing.T, entities ...interface{}) (*gorm.DB, func()) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	_ = db.Exec("PRAGMA busy_timeout = 5000;")
	_ = db.Exec("PRAGMA journal_mode = WAL;")

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB: %v", err)
	}

	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(time.Minute)

	if err := db.AutoMigrate(entities...); err != nil {
		t.Fatalf("failed to automigrate: %v", err)
	}

	cleanup := func() {
		_ = sqlDB.Close()
	}

	return db, cleanup
}
