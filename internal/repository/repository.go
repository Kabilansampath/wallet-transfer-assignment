package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"wallet-transfer/internal/models"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateIdempotencyRecord(ctx context.Context, tx *gorm.DB, record *models.IdempotencyRecord) error {
	return tx.WithContext(ctx).Create(record).Error
}

func (r *Repository) GetIdempotencyRecordForUpdate(ctx context.Context, tx *gorm.DB, key string) (*models.IdempotencyRecord, error) {
	var record models.IdempotencyRecord
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("idempotency_key = ?", key).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *Repository) UpdateIdempotencyRecord(ctx context.Context, tx *gorm.DB, record *models.IdempotencyRecord) error {
	return tx.WithContext(ctx).Save(record).Error
}

func (r *Repository) CreateTransfer(ctx context.Context, tx *gorm.DB, transfer *models.Transfer) error {
	return tx.WithContext(ctx).Create(transfer).Error
}

func (r *Repository) UpdateTransfer(ctx context.Context, tx *gorm.DB, transfer *models.Transfer) error {
	return tx.WithContext(ctx).Save(transfer).Error
}

func (r *Repository) LockWalletsInOrder(ctx context.Context, tx *gorm.DB, firstID, secondID string) ([]models.Wallet, error) {
	var wallets []models.Wallet
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", []string{firstID, secondID}).
		Order("id ASC").
		Find(&wallets).Error
	if err != nil {
		return nil, err
	}
	return wallets, nil
}

func (r *Repository) UpdateWallet(ctx context.Context, tx *gorm.DB, wallet *models.Wallet) error {
	return tx.WithContext(ctx).Save(wallet).Error
}

func (r *Repository) CreateLedgerEntries(ctx context.Context, tx *gorm.DB, entries []models.LedgerEntry) error {
	return tx.WithContext(ctx).Create(&entries).Error
}
