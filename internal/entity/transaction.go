package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TransactionStatus string

const (
	TransactionStatusPending  TransactionStatus = "pending"
	TransactionStatusApproved TransactionStatus = "approved"
	TransactionStatusDeclined TransactionStatus = "declined"
)

type Transaction struct {
	ID            uuid.UUID         `gorm:"type:uuid;primaryKey"`
	MerchantID    uuid.UUID         `gorm:"type:uuid;not null;index"`
	Amount        decimal.Decimal   `gorm:"type:numeric(20,2);not null"`
	Currency      string            `gorm:"type:varchar(3);not null;default:'BRL'"`
	Status        TransactionStatus `gorm:"type:varchar(20);not null;default:'pending'"`
	Description   string            `gorm:"type:text"`
	PaymentMethod string            `gorm:"type:varchar(50);not null"`
	Metadata      JSON              `gorm:"type:jsonb"`
	CreatedAt     time.Time         `gorm:"not null"`
	UpdatedAt     time.Time         `gorm:"not null"`
	DeletedAt     gorm.DeletedAt    `gorm:"index"`
}

func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (Transaction) TableName() string {
	return "transactions"
}
