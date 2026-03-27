package benchmark

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"poc-testcontainers/internal/entity"
	"poc-testcontainers/internal/repository"
	"poc-testcontainers/tests/testdb"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

var shared *testdb.SharedPostgres

// TestMain é o ponto central pra avaliar impacto no CI.
// Sobe 1 container, roda todos os testes da suite, derruba.
func TestMain(m *testing.M) {
	shared = testdb.NewSharedPostgres()

	if err := shared.Start(); err != nil {
		fmt.Printf("FATAL: falha ao subir container: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== CONTAINER STARTUP: %s ===\n\n", shared.StartupTime.Round(time.Millisecond))

	code := m.Run()

	shared.Stop()
	os.Exit(code)
}

// TestShared_MultipleTests simula uma suite com vários testes usando o mesmo container.
// Isso é o cenário real: N testes de repositório rodando no CI.
func TestShared_MultipleTests(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T, repo *repository.TransactionRepository)
	}{
		{
			name: "Create",
			fn: func(t *testing.T, repo *repository.TransactionRepository) {
				tx := &entity.Transaction{
					MerchantID:    uuid.New(),
					Amount:        decimal.NewFromFloat(100),
					Status:        entity.TransactionStatusPending,
					PaymentMethod: "credit_card",
				}
				require.NoError(t, repo.Create(context.Background(), tx))
			},
		},
		{
			name: "CreateAndFind",
			fn: func(t *testing.T, repo *repository.TransactionRepository) {
				ctx := context.Background()
				tx := &entity.Transaction{
					MerchantID:    uuid.New(),
					Amount:        decimal.NewFromFloat(250.75),
					Status:        entity.TransactionStatusApproved,
					PaymentMethod: "pix",
				}
				require.NoError(t, repo.Create(ctx, tx))
				found, err := repo.FindByID(ctx, tx.ID)
				require.NoError(t, err)
				require.Equal(t, tx.ID, found.ID)
			},
		},
		{
			name: "SumByMerchant",
			fn: func(t *testing.T, repo *repository.TransactionRepository) {
				ctx := context.Background()
				merchantID := uuid.New()
				for i := 0; i < 10; i++ {
					tx := &entity.Transaction{
						MerchantID:    merchantID,
						Amount:        decimal.NewFromFloat(100),
						Status:        entity.TransactionStatusApproved,
						PaymentMethod: "credit_card",
					}
					require.NoError(t, repo.Create(ctx, tx))
				}
				total, err := repo.SumByMerchant(ctx, merchantID)
				require.NoError(t, err)
				require.True(t, decimal.NewFromFloat(1000).Equal(total))
			},
		},
		{
			name: "JSONB_Query",
			fn: func(t *testing.T, repo *repository.TransactionRepository) {
				ctx := context.Background()
				tx := &entity.Transaction{
					MerchantID:    uuid.New(),
					Amount:        decimal.NewFromFloat(50),
					Status:        entity.TransactionStatusApproved,
					PaymentMethod: "pix",
					Metadata:      entity.JSON{"source": "api", "version": "v2"},
				}
				require.NoError(t, repo.Create(ctx, tx))
				results, err := repo.FindByMetadata(ctx, "source", "api")
				require.NoError(t, err)
				require.Len(t, results, 1)
			},
		},
		{
			name: "BulkInsert_50records",
			fn: func(t *testing.T, repo *repository.TransactionRepository) {
				ctx := context.Background()
				merchantID := uuid.New()
				for i := 0; i < 50; i++ {
					tx := &entity.Transaction{
						MerchantID:    merchantID,
						Amount:        decimal.NewFromFloat(float64(i) + 1),
						Status:        entity.TransactionStatusApproved,
						PaymentMethod: "credit_card",
					}
					require.NoError(t, repo.Create(ctx, tx))
				}
				txs, err := repo.FindByMerchantID(ctx, merchantID)
				require.NoError(t, err)
				require.Len(t, txs, 50)
			},
		},
	}

	suiteStart := time.Now()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			db := shared.NewDB(t, &entity.Transaction{})
			repo := repository.NewTransactionRepository(db)
			tc.fn(t, repo)
			t.Logf("duration: %s", time.Since(start).Round(time.Millisecond))
		})
	}

	t.Logf("\n=== SUITE TOTAL (5 testes): %s ===", time.Since(suiteStart).Round(time.Millisecond))
	t.Logf("=== CONTAINER STARTUP foi: %s ===", shared.StartupTime.Round(time.Millisecond))
	t.Logf("=== TEMPO TOTAL (startup + testes): %s ===", (shared.StartupTime + time.Since(suiteStart)).Round(time.Millisecond))
}
