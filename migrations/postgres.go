package migrations

import (
	"fmt"
	"wallet-transfer/internal/models"

	"gorm.io/gorm"
)

func SeedWallets(db *gorm.DB) {
	var count int64

	db.Model(&models.Wallet{}).Where("id IN ?", []string{"wallet_1", "wallet_2"}).Count(&count)

	if count == 0 {
		wallets := []models.Wallet{
			{ID: "wallet_1", Balance: 1000},
			{ID: "wallet_2", Balance: 500},
		}

		for _, w := range wallets {
			db.Create(&w)
		}

		fmt.Println("✅ Default wallets created")
	}
}
