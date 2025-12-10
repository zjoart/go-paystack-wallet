package wallet

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	CreateWallet(wallet *Wallet) error
	GetWalletByUserID(userID string) (*Wallet, error)
	GetWalletByNumber(number string) (*Wallet, error)
	CreditWallet(walletID string, amount int64) error
	DebitWallet(walletID string, amount int64) error

	CreateTransaction(tx *Transaction) error
	GetTransactionByReference(ref string) (*Transaction, error)
	UpdateTransactionStatus(ref string, status TransactionStatus) error
	GetTransactions(walletID string, limit, offset int) ([]Transaction, error)
	CountTransactions(walletID string) (int64, error)
	TransferFunds(fromID, toID, senderNumber, recipientNumber, reference string, amount int64, description string) error
	ProcessDeposit(reference string, amount int64) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) TransferFunds(fromID, toID, senderNumber, recipientNumber, reference string, amount int64, description string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {

		//debit initiator
		res := tx.Model(&Wallet{}).
			Where("id = ? AND balance >= ?", fromID, amount).
			UpdateColumn("balance", gorm.Expr("balance - ?", amount))

		if res.Error != nil {
			return res.Error
		}

		if res.RowsAffected == 0 {
			return errors.New("insufficient balance")
		}

		// credit recipient
		if err := tx.Model(&Wallet{}).Where("id = ?", toID).UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
			return err
		}

		// create sender debit transaction record
		senderTx := Transaction{
			WalletID:              uuid.MustParse(fromID),
			Reference:             reference + "-debit",
			Category:              CategoryTransfer,
			Type:                  TransactionDebit,
			Amount:                amount,
			Status:                TransactionSuccess,
			SenderWalletNumber:    &senderNumber,
			RecipientWalletNumber: &recipientNumber,
			Description:           description,
		}

		if err := tx.Create(&senderTx).Error; err != nil {
			return err
		}

		// create recipient credit transaction record
		recipientTx := Transaction{
			WalletID:              uuid.MustParse(toID),
			Reference:             reference + "-credit",
			Category:              CategoryTransfer,
			Type:                  TransactionCredit,
			Amount:                amount,
			Status:                TransactionSuccess,
			SenderWalletNumber:    &senderNumber,
			RecipientWalletNumber: &recipientNumber,
			Description:           description,
		}

		if err := tx.Create(&recipientTx).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *repository) CreateWallet(wallet *Wallet) error {
	return r.db.Create(wallet).Error
}

func (r *repository) GetWalletByUserID(userID string) (*Wallet, error) {
	var wallet Wallet
	if err := r.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *repository) GetWalletByNumber(number string) (*Wallet, error) {
	var wallet Wallet
	if err := r.db.Where("wallet_number = ?", number).First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *repository) CreditWallet(walletID string, amount int64) error {
	return r.db.Model(&Wallet{}).
		Where("id = ?", walletID).
		UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error
}

func (r *repository) DebitWallet(walletID string, amount int64) error {
	result := r.db.Model(&Wallet{}).
		Where("id = ? AND balance >= ?", walletID, amount).
		UpdateColumn("balance", gorm.Expr("balance - ?", amount))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("insufficient balance")
	}
	return nil
}

func (r *repository) CreateTransaction(tx *Transaction) error {
	return r.db.Create(tx).Error
}

func (r *repository) GetTransactionByReference(ref string) (*Transaction, error) {
	var tx Transaction
	if err := r.db.Where("reference = ?", ref).First(&tx).Error; err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *repository) UpdateTransactionStatus(ref string, status TransactionStatus) error {
	return r.db.Model(&Transaction{}).
		Where("reference = ?", ref).
		Update("status", status).Error
}

func (r *repository) GetTransactions(walletID string, limit, offset int) ([]Transaction, error) {
	var txs []Transaction
	err := r.db.Where("wallet_id = ?", walletID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&txs).Error
	return txs, err
}

func (r *repository) CountTransactions(walletID string) (int64, error) {
	var count int64
	err := r.db.Model(&Transaction{}).Where("wallet_id = ?", walletID).Count(&count).Error
	return count, err
}

func (r *repository) ProcessDeposit(reference string, amount int64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var transaction Transaction
		if err := tx.Where("reference = ?", reference).First(&transaction).Error; err != nil {
			return err
		}

		if transaction.Status == TransactionSuccess {
			return nil
		}

		if err := tx.Model(&Wallet{}).Where("id = ?", transaction.WalletID).UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
			return err
		}

		if err := tx.Model(&Transaction{}).Where("reference = ?", reference).Update("status", TransactionSuccess).Error; err != nil {
			return err
		}

		return nil
	})
}
