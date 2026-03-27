package repository

import (
	"context"
	"fmt"
	"time"

	"poc-testcontainers/internal/entity"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, tx *entity.Transaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *TransactionRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Transaction, error) {
	var tx entity.Transaction
	err := r.db.WithContext(ctx).First(&tx, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *TransactionRepository) FindByMerchantID(ctx context.Context, merchantID uuid.UUID) ([]entity.Transaction, error) {
	var txs []entity.Transaction
	err := r.db.WithContext(ctx).Where("merchant_id = ?", merchantID).Find(&txs).Error
	return txs, err
}

// SumByMerchant usa funções SQL que diferem entre SQLite e Postgres.
// No Postgres: COALESCE + casting funciona nativamente com numeric.
// No SQLite: numeric não existe, e o comportamento de COALESCE pode diferir.
func (r *TransactionRepository) SumByMerchant(ctx context.Context, merchantID uuid.UUID) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}

	err := r.db.WithContext(ctx).
		Model(&entity.Transaction{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("merchant_id = ? AND status = ?", merchantID, entity.TransactionStatusApproved).
		Where("deleted_at IS NULL").
		Scan(&result).Error

	return result.Total, err
}

// FindByMetadata usa operador JSONB do Postgres (@>), que NÃO existe no SQLite.
// Este é o caso mais claro de incompatibilidade.
func (r *TransactionRepository) FindByMetadata(ctx context.Context, key, value string) ([]entity.Transaction, error) {
	var txs []entity.Transaction

	query := fmt.Sprintf(`metadata @> '{ "%s": "%s" }'`, key, value)
	err := r.db.WithContext(ctx).Where(query).Find(&txs).Error
	return txs, err
}

// RecentApproved usa interval do Postgres, que não existe no SQLite.
func (r *TransactionRepository) RecentApproved(ctx context.Context, since time.Duration) ([]entity.Transaction, error) {
	var txs []entity.Transaction

	err := r.db.WithContext(ctx).
		Where("status = ? AND created_at > NOW() - INTERVAL '1 hour'", entity.TransactionStatusApproved).
		Find(&txs).Error

	return txs, err
}
