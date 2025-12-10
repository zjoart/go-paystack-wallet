package wallet

import (
	"time"

	"github.com/google/uuid"
)

type Wallet struct {
	ID           uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	WalletNumber string    `gorm:"uniqueIndex;not null" json:"wallet_number"`
	Balance      int64     `gorm:"not null;default:0" json:"balance"`
	Currency     string    `gorm:"not null;default:NGN" json:"currency"`
	PinHash      string    `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type TransactionType string

const (
	TransactionDeposit    TransactionType = "DEPOSIT"
	TransactionWithdrawal TransactionType = "WITHDRAWAL"
	TransactionTransfer   TransactionType = "TRANSFER"
)

type TransactionStatus string

const (
	TransactionPending TransactionStatus = "PENDING"
	TransactionSuccess TransactionStatus = "SUCCESS"
	TransactionFailed  TransactionStatus = "FAILED"
)

type Transaction struct {
	ID                    uuid.UUID         `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"id"`
	WalletID              uuid.UUID         `gorm:"type:uuid;not null" json:"wallet_id"`
	Reference             string            `gorm:"uniqueIndex;not null" json:"reference"`
	Type                  TransactionType   `gorm:"not null" json:"type"`
	Amount                int64             `gorm:"not null" json:"amount"`
	Status                TransactionStatus `gorm:"not null" json:"status"`
	SenderWalletNumber    *string           `json:"sender_wallet_number,omitempty"`
	RecipientWalletNumber *string           `json:"recipient_wallet_number,omitempty"`
	Description           string            `json:"description"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
}
