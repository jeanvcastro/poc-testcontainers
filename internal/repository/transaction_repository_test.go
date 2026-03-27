package repository

import (
	"context"
	"fmt"
	"os"
	"testing"

	"poc-testcontainers/internal/entity"
	"poc-testcontainers/tests/testdb"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var shared *testdb.SharedPostgres

func TestMain(m *testing.M) {
	shared = testdb.NewSharedPostgres()
	if err := shared.Start(); err != nil {
		fmt.Printf("FATAL: falha ao subir container: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n=== CONTAINER STARTUP: %s ===\n\n", shared.StartupTime.Round(1e6))

	code := m.Run()

	shared.Stop()
	os.Exit(code)
}

// =============================================================================
// Testes com Testcontainers (Postgres real) — container compartilhado
// =============================================================================

func TestPostgres_CreateAndFind(t *testing.T) {
	db := shared.NewDB(t, &entity.Transaction{})
	repo := NewTransactionRepository(db)
	ctx := context.Background()
	merchantID := uuid.New()

	tx := &entity.Transaction{
		MerchantID:    merchantID,
		Amount:        decimal.NewFromFloat(150.50),
		Currency:      "BRL",
		Status:        entity.TransactionStatusApproved,
		PaymentMethod: "credit_card",
		Description:   "Compra teste",
		Metadata:      entity.JSON{"order_id": "ORD-123", "installments": "3"},
	}

	err := repo.Create(ctx, tx)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, tx.ID)

	found, err := repo.FindByID(ctx, tx.ID)
	require.NoError(t, err)
	assert.Equal(t, tx.ID, found.ID)
	assert.Equal(t, merchantID, found.MerchantID)
	assert.True(t, decimal.NewFromFloat(150.50).Equal(found.Amount))
}

func TestPostgres_SumByMerchant(t *testing.T) {
	db := shared.NewDB(t, &entity.Transaction{})
	repo := NewTransactionRepository(db)
	ctx := context.Background()
	merchantID := uuid.New()

	transactions := []*entity.Transaction{
		{MerchantID: merchantID, Amount: decimal.NewFromFloat(100.00), Status: entity.TransactionStatusApproved, PaymentMethod: "credit_card"},
		{MerchantID: merchantID, Amount: decimal.NewFromFloat(200.50), Status: entity.TransactionStatusApproved, PaymentMethod: "credit_card"},
		{MerchantID: merchantID, Amount: decimal.NewFromFloat(50.00), Status: entity.TransactionStatusDeclined, PaymentMethod: "credit_card"}, // não soma
	}

	for _, tx := range transactions {
		require.NoError(t, repo.Create(ctx, tx))
	}

	total, err := repo.SumByMerchant(ctx, merchantID)
	require.NoError(t, err)
	assert.True(t, decimal.NewFromFloat(300.50).Equal(total))
}

// TestPostgres_FindByMetadata_JSONB demonstra query com operador @> do Postgres.
// Este teste FALHA com SQLite porque @> não existe lá.
func TestPostgres_FindByMetadata_JSONB(t *testing.T) {
	db := shared.NewDB(t, &entity.Transaction{})
	repo := NewTransactionRepository(db)
	ctx := context.Background()

	tx := &entity.Transaction{
		MerchantID:    uuid.New(),
		Amount:        decimal.NewFromFloat(99.90),
		Status:        entity.TransactionStatusApproved,
		PaymentMethod: "pix",
		Metadata:      entity.JSON{"channel": "mobile", "app_version": "2.1.0"},
	}
	require.NoError(t, repo.Create(ctx, tx))

	results, err := repo.FindByMetadata(ctx, "channel", "mobile")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, tx.ID, results[0].ID)
}

// =============================================================================
// Teste comparativo: mesmo cenário, SQLite falha
// =============================================================================

func TestSQLite_FindByMetadata_JSONB_FAILS(t *testing.T) {
	db, cleanup := testdb.NewSQLiteDB(t, &entity.Transaction{})
	defer cleanup()

	repo := NewTransactionRepository(db)
	ctx := context.Background()

	tx := &entity.Transaction{
		MerchantID:    uuid.New(),
		Amount:        decimal.NewFromFloat(99.90),
		Status:        entity.TransactionStatusApproved,
		PaymentMethod: "pix",
		Metadata:      entity.JSON{"channel": "mobile"},
	}
	require.NoError(t, repo.Create(ctx, tx))

	// Esta query usa @> que não existe no SQLite → ERRO
	_, err := repo.FindByMetadata(ctx, "channel", "mobile")
	assert.Error(t, err, "SQLite não suporta operador JSONB @> — este teste demonstra a incompatibilidade")
}
