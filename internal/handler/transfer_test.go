package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"wallet-transfer/internal/handler"
	"wallet-transfer/internal/models"
	"wallet-transfer/internal/repository"
	"wallet-transfer/internal/service"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf(
		"host=%s user=%s dbname=%s password=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_USERNAME", "postgres"),
		getEnv("DB_NAME", "wallet_transfer_test"),
		getEnv("DB_PASSWORD", "postgres"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect test db: %v", err)
	}

	err = db.AutoMigrate(
		&models.Wallet{},
		&models.Transfer{},
		&models.LedgerEntry{},
		&models.IdempotencyRecord{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test db: %v", err)
	}

	cleanupTables(t, db)
	seedWallets(t, db)

	repo := repository.NewRepository(db)
	svc := service.NewTransferService(db, repo)
	h := handler.NewTransferHandler(svc)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/transfers", h.CreateTransfer)

	return r, db
}

func cleanupTables(t *testing.T, db *gorm.DB) {
	t.Helper()

	db.Exec("DELETE FROM ledger_entries")
	db.Exec("DELETE FROM idempotency_records")
	db.Exec("DELETE FROM transfers")
	db.Exec("DELETE FROM wallets")
}

func seedWallets(t *testing.T, db *gorm.DB) {
	t.Helper()

	wallets := []models.Wallet{
		{ID: "wallet_1", Balance: 1000},
		{ID: "wallet_2", Balance: 500},
	}

	for _, w := range wallets {
		if err := db.Create(&w).Error; err != nil {
			t.Fatalf("failed to seed wallet: %v", err)
		}
	}
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func TestCreateTransfer_Success(t *testing.T) {
	r, db := setupTestRouter(t)

	body := map[string]any{
		"idempotencyKey": "key-success",
		"fromWalletId":   "wallet_1",
		"toWalletId":     "wallet_2",
		"amount":         100,
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var fromWallet models.Wallet
	var toWallet models.Wallet

	db.First(&fromWallet, "id = ?", "wallet_1")
	db.First(&toWallet, "id = ?", "wallet_2")

	if fromWallet.Balance != 900 {
		t.Fatalf("expected wallet_1 balance 900, got %d", fromWallet.Balance)
	}
	if toWallet.Balance != 600 {
		t.Fatalf("expected wallet_2 balance 600, got %d", toWallet.Balance)
	}

	var ledgerCount int64
	db.Model(&models.LedgerEntry{}).Count(&ledgerCount)
	if ledgerCount != 2 {
		t.Fatalf("expected 2 ledger entries, got %d", ledgerCount)
	}
}

func TestCreateTransfer_IdempotencyReplay(t *testing.T) {
	r, db := setupTestRouter(t)

	body := map[string]any{
		"idempotencyKey": "key-replay",
		"fromWalletId":   "wallet_1",
		"toWalletId":     "wallet_2",
		"amount":         100,
	}

	jsonBody, _ := json.Marshal(body)

	req1, _ := http.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	req2, _ := http.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
		t.Fatalf("expected both responses 200, got %d and %d", w1.Code, w2.Code)
	}

	var transferCount int64
	db.Model(&models.Transfer{}).Count(&transferCount)
	if transferCount != 1 {
		t.Fatalf("expected 1 transfer, got %d", transferCount)
	}
}

func TestCreateTransfer_IdempotencyPayloadMismatch(t *testing.T) {
	r, _ := setupTestRouter(t)

	body1 := map[string]any{
		"idempotencyKey": "same-key",
		"fromWalletId":   "wallet_1",
		"toWalletId":     "wallet_2",
		"amount":         100,
	}

	body2 := map[string]any{
		"idempotencyKey": "same-key",
		"fromWalletId":   "wallet_1",
		"toWalletId":     "wallet_2",
		"amount":         200,
	}

	jsonBody1, _ := json.Marshal(body1)
	req1, _ := http.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(jsonBody1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	jsonBody2, _ := json.Marshal(body2)
	req2, _ := http.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(jsonBody2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d, body=%s", w2.Code, w2.Body.String())
	}
}

func TestCreateTransfer_InsufficientBalance(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := map[string]any{
		"idempotencyKey": "key-insufficient",
		"fromWalletId":   "wallet_1",
		"toWalletId":     "wallet_2",
		"amount":         5000,
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestCreateTransfer_WalletNotFound(t *testing.T) {
	r, _ := setupTestRouter(t)

	body := map[string]any{
		"idempotencyKey": "key-wallet-not-found",
		"fromWalletId":   "wallet_x",
		"toWalletId":     "wallet_2",
		"amount":         100,
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d, body=%s", w.Code, w.Body.String())
	}
}