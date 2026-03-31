package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"wallet-transfer/internal/models"
	"wallet-transfer/internal/repository"
)

var (
	ErrIdempotencyConflict = errors.New("idempotency key reused with different payload")
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrInsufficientFunds   = errors.New("insufficient funds")
	ErrSameWallet          = errors.New("fromWalletId and toWalletId must be different")
)

type TransferService struct {
	db   *gorm.DB
	repo *repository.Repository
}

func NewTransferService(db *gorm.DB, repo *repository.Repository) *TransferService {
	return &TransferService{db: db, repo: repo}
}

type Result struct {
	HTTPCode int
	Body     any
}

func (s *TransferService) CreateTransfer(ctx context.Context, req models.TransferRequest) (Result, error) {
	if req.FromWalletID == req.ToWalletID {
		return Result{HTTPCode: 400, Body: models.ErrorResponse{Error: ErrSameWallet.Error()}}, nil
	}

	requestHash, err := hashRequest(req)
	if err != nil {
		return Result{}, err
	}

	var result Result
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record, err := s.repo.GetIdempotencyRecordForUpdate(ctx, tx, req.IdempotencyKey)
		if err == nil {
			if record.RequestHash != requestHash {
				result = Result{HTTPCode: 409, Body: models.ErrorResponse{Error: ErrIdempotencyConflict.Error()}}
				return nil
			}
			result = Result{HTTPCode: record.ResponseCode, Body: json.RawMessage(record.ResponseBody)}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		record = &models.IdempotencyRecord{
			IdempotencyKey: req.IdempotencyKey,
			RequestHash:    requestHash,
			Status:         models.IdempotencyStarted,
			ResponseCode:   202,
			ResponseBody:   `{"status":"PENDING"}`,
		}
		if err := s.repo.CreateIdempotencyRecord(ctx, tx, record); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				locked, lockErr := s.repo.GetIdempotencyRecordForUpdate(ctx, tx, req.IdempotencyKey)
				if lockErr != nil {
					return lockErr
				}
				if locked.RequestHash != requestHash {
					result = Result{HTTPCode: 409, Body: models.ErrorResponse{Error: ErrIdempotencyConflict.Error()}}
					return nil
				}
				result = Result{HTTPCode: locked.ResponseCode, Body: json.RawMessage(locked.ResponseBody)}
				return nil
			}
			return err
		}

		transfer := &models.Transfer{
			IdempotencyKey: req.IdempotencyKey,
			FromWalletID:   req.FromWalletID,
			ToWalletID:     req.ToWalletID,
			Amount:         req.Amount,
			Status:         models.TransferPending,
		}
		if err := s.repo.CreateTransfer(ctx, tx, transfer); err != nil {
			return err
		}

		firstID, secondID := orderedWalletIDs(req.FromWalletID, req.ToWalletID)
		wallets, err := s.repo.LockWalletsInOrder(ctx, tx, firstID, secondID)
		if err != nil {
			return err
		}
		if len(wallets) != 2 {
			transfer.Status = models.TransferFailed
			reason := ErrWalletNotFound.Error()
			transfer.FailureReason = &reason
			if err := s.repo.UpdateTransfer(ctx, tx, transfer); err != nil {
				return err
			}
			resp := models.ErrorResponse{Error: ErrWalletNotFound.Error()}
			body, _ := json.Marshal(resp)
			record.Status = models.IdempotencyCompleted
			record.TransferID = &transfer.ID
			record.ResponseCode = 404
			record.ResponseBody = string(body)
			if err := s.repo.UpdateIdempotencyRecord(ctx, tx, record); err != nil {
				return err
			}
			result = Result{HTTPCode: 404, Body: resp}
			return nil
		}

		walletMap := make(map[string]*models.Wallet, 2)
		for i := range wallets {
			walletMap[wallets[i].ID] = &wallets[i]
		}
		fromWallet := walletMap[req.FromWalletID]
		toWallet := walletMap[req.ToWalletID]

		if fromWallet.Balance < req.Amount {
			transfer.Status = models.TransferFailed
			reason := ErrInsufficientFunds.Error()
			transfer.FailureReason = &reason
			if err := s.repo.UpdateTransfer(ctx, tx, transfer); err != nil {
				return err
			}
			resp := models.ErrorResponse{Error: ErrInsufficientFunds.Error()}
			body, _ := json.Marshal(resp)
			record.Status = models.IdempotencyCompleted
			record.TransferID = &transfer.ID
			record.ResponseCode = 409
			record.ResponseBody = string(body)
			if err := s.repo.UpdateIdempotencyRecord(ctx, tx, record); err != nil {
				return err
			}
			result = Result{HTTPCode: 409, Body: resp}
			return nil
		}

		fromWallet.Balance -= req.Amount
		toWallet.Balance += req.Amount
		if err := s.repo.UpdateWallet(ctx, tx, fromWallet); err != nil {
			return err
		}
		if err := s.repo.UpdateWallet(ctx, tx, toWallet); err != nil {
			return err
		}

		entries := []models.LedgerEntry{
			{WalletID: req.FromWalletID, TransferID: transfer.ID, Type: models.LedgerDebit, Amount: req.Amount},
			{WalletID: req.ToWalletID, TransferID: transfer.ID, Type: models.LedgerCredit, Amount: req.Amount},
		}
		if err := s.repo.CreateLedgerEntries(ctx, tx, entries); err != nil {
			return err
		}

		transfer.Status = models.TransferProcessed
		if err := s.repo.UpdateTransfer(ctx, tx, transfer); err != nil {
			return err
		}

		resp := models.TransferResponse{
			TransferID:         transfer.ID,
			Status:             string(transfer.Status),
			FromWalletID:       transfer.FromWalletID,
			ToWalletID:         transfer.ToWalletID,
			Amount:             transfer.Amount,
			SourceBalance:      fromWallet.Balance,
			DestinationBalance: toWallet.Balance,
		}
		body, _ := json.Marshal(resp)
		record.Status = models.IdempotencyCompleted
		record.TransferID = &transfer.ID
		record.ResponseCode = 200
		record.ResponseBody = string(body)
		if err := s.repo.UpdateIdempotencyRecord(ctx, tx, record); err != nil {
			return err
		}
		result = Result{HTTPCode: 200, Body: resp}
		return nil
	})
	if err != nil {
		return Result{}, err
	}

	if raw, ok := result.Body.(json.RawMessage); ok {
		var success models.TransferResponse
		if err := json.Unmarshal(raw, &success); err == nil && success.TransferID != 0 {
			return Result{HTTPCode: result.HTTPCode, Body: success}, nil
		}
		var fail models.ErrorResponse
		if err := json.Unmarshal(raw, &fail); err == nil && fail.Error != "" {
			return Result{HTTPCode: result.HTTPCode, Body: fail}, nil
		}
	}

	return result, nil
}

func hashRequest(req models.TransferRequest) (string, error) {
	normalized := struct {
		FromWalletID string `json:"fromWalletId"`
		ToWalletID   string `json:"toWalletId"`
		Amount       int64  `json:"amount"`
	}{
		FromWalletID: req.FromWalletID,
		ToWalletID:   req.ToWalletID,
		Amount:       req.Amount,
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func orderedWalletIDs(a, b string) (string, string) {
	if a <= b {
		return a, b
	}
	return b, a
}

// Silence unused imports when GORM drivers change lock behavior across versions.
var _ = clause.Locking{}
