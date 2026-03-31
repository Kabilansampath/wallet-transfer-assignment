package models

import "time"

type TransferStatus string

type LedgerEntryType string

type IdempotencyStatus string

const (
	TransferPending   TransferStatus = "PENDING"
	TransferProcessed TransferStatus = "PROCESSED"
	TransferFailed    TransferStatus = "FAILED"

	LedgerDebit  LedgerEntryType = "DEBIT"
	LedgerCredit LedgerEntryType = "CREDIT"

	IdempotencyStarted   IdempotencyStatus = "STARTED"
	IdempotencyCompleted IdempotencyStatus = "COMPLETED"
)

type Wallet struct {
	ID        string    `gorm:"type:text;primaryKey"`
	Balance   int64     `gorm:"not null;check:balance_non_negative,balance >= 0"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

type Transfer struct {
	ID             uint64         `gorm:"primaryKey;autoIncrement"`
	IdempotencyKey string         `gorm:"type:text;not null;uniqueIndex"`
	FromWalletID   string         `gorm:"type:text;not null;index"`
	ToWalletID     string         `gorm:"type:text;not null;index"`
	Amount         int64          `gorm:"not null;check:amount_positive,amount > 0"`
	Status         TransferStatus `gorm:"type:text;not null;index"`
	FailureReason  *string        `gorm:"type:text"`
	CreatedAt      time.Time      `gorm:"not null;default:now()"`
	UpdatedAt      time.Time      `gorm:"not null;default:now()"`
}

type LedgerEntry struct {
	ID         uint64          `gorm:"primaryKey;autoIncrement"`
	WalletID    string          `gorm:"type:text;not null;index"`
	TransferID  uint64          `gorm:"not null;index"`
	Type        LedgerEntryType `gorm:"type:text;not null"`
	Amount      int64           `gorm:"not null;check:ledger_amount_positive,amount > 0"`
	CreatedAt   time.Time       `gorm:"not null;default:now()"`
}

type IdempotencyRecord struct {
	ID              uint64            `gorm:"primaryKey;autoIncrement"`
	IdempotencyKey  string            `gorm:"type:text;not null;uniqueIndex"`
	RequestHash     string            `gorm:"type:text;not null"`
	Status          IdempotencyStatus `gorm:"type:text;not null;index"`
	ResponseCode    int               `gorm:"not null"`
	ResponseBody    string            `gorm:"type:text;not null"`
	TransferID      *uint64           `gorm:"index"`
	CreatedAt       time.Time         `gorm:"not null;default:now()"`
	UpdatedAt       time.Time         `gorm:"not null;default:now()"`
}

type TransferRequest struct {
	IdempotencyKey string `json:"idempotencyKey" binding:"required"`
	FromWalletID   string `json:"fromWalletId" binding:"required"`
	ToWalletID     string `json:"toWalletId" binding:"required"`
	Amount         int64  `json:"amount" binding:"required,gt=0"`
}

type TransferResponse struct {
	TransferID      uint64 `json:"transferId"`
	Status          string `json:"status"`
	FromWalletID    string `json:"fromWalletId"`
	ToWalletID      string `json:"toWalletId"`
	Amount          int64  `json:"amount"`
	SourceBalance   int64  `json:"sourceBalance"`
	DestinationBalance int64 `json:"destinationBalance"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
