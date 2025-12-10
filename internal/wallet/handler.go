package wallet

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/zjoart/go-paystack-wallet/internal/user"
	"github.com/zjoart/go-paystack-wallet/pkg/config"
	"github.com/zjoart/go-paystack-wallet/pkg/logger"
	"github.com/zjoart/go-paystack-wallet/pkg/utils"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	Config config.Config
	Repo   Repository
}

func NewHandler(cfg config.Config, repo Repository) *Handler {
	return &Handler{Config: cfg, Repo: repo}
}

type CreateWalletRequest struct {
	Pin string `json:"pin"`
}

func (h *Handler) CreateWallet(w http.ResponseWriter, r *http.Request) {
	usr, ok := r.Context().Value(utils.UserKey).(user.User)
	if !ok {
		utils.BuildErrorResponse(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	var req CreateWalletRequest
	if status, err := utils.DecodeJSONBody(w, r, &req); err != nil {
		utils.BuildErrorResponse(w, status, "Invalid request", map[string]string{"error": err.Error()})
		return
	}

	if len(req.Pin) != 4 {
		utils.BuildErrorResponse(w, http.StatusBadRequest, "PIN must be 4 digits", nil)
		return
	}

	// check if wallet exists
	existingWallet, _ := h.Repo.GetWalletByUserID(usr.ID.String())
	if existingWallet != nil {
		utils.BuildErrorResponse(w, http.StatusConflict, "User already has a wallet", nil)
		return
	}

	hashedPin, err := bcrypt.GenerateFromPassword([]byte(req.Pin), bcrypt.DefaultCost)
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to secure PIN", nil)
		return
	}

	wallet := Wallet{
		UserID:       usr.ID,
		WalletNumber: generateWalletNumber(),
		PinHash:      string(hashedPin),
		Balance:      0,
		Currency:     "NGN",
	}

	if err := h.Repo.CreateWallet(&wallet); err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to create wallet", nil)
		return
	}

	utils.BuildSuccessResponse(w, http.StatusCreated, "Wallet created successfully", map[string]interface{}{
		"wallet_number": wallet.WalletNumber,
		"balance":       wallet.Balance,
		"currency":      wallet.Currency,
	})
}

type DepositRequest struct {
	Amount int64 `json:"amount"` // in Kobo
}

func (h *Handler) WalletDeposit(w http.ResponseWriter, r *http.Request) {

	usr, _ := r.Context().Value(utils.UserKey).(user.User)

	var req DepositRequest
	if status, err := utils.DecodeJSONBody(w, r, &req); err != nil {
		utils.BuildErrorResponse(w, status, "Invalid request", map[string]string{"error": err.Error()})
		return
	}

	if req.Amount < h.Config.MinTransactionAmount {
		utils.BuildErrorResponse(w, http.StatusBadRequest, "Invalid amount, can't be less than 100 Naira (10000 Kobo)", nil)
		return
	}

	wallet, err := h.Repo.GetWalletByUserID(usr.ID.String())
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusNotFound, "Wallet not found", nil)
		return
	}

	paystackUrl := "https://api.paystack.co/transaction/initialize"
	reference := fmt.Sprintf("dep-%s-%d", usr.ID.String(), time.Now().UnixNano())

	payload := map[string]interface{}{
		"email":        usr.Email,
		"amount":       req.Amount,
		"reference":    reference,
		"currency":     "NGN",
		"channels":     h.Config.PaystackChannels,
		"callback_url": fmt.Sprintf("%s/api/wallet/deposit/callback", h.Config.Host),
		"metadata":     map[string]interface{}{"wallet_id": wallet.ID.String()},
	}

	jsonPayload, _ := json.Marshal(payload)

	paystackReq, _ := http.NewRequest("POST", paystackUrl, strings.NewReader(string(jsonPayload)))
	paystackReq.Header.Set("Authorization", "Bearer "+h.Config.PaystackSecret)
	paystackReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(paystackReq)
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to reach Paystack", nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		logger.Error("Paystack error", logger.Fields{
			"status_code": resp.StatusCode,
			"body":        string(respBody),
			"payload":     payload,
		})
		utils.BuildErrorResponse(w, http.StatusBadGateway, "Paystack error", nil)
		return
	}

	var paystackResp struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			AuthorizationURL string `json:"authorization_url"`
			Reference        string `json:"reference"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&paystackResp); err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to parse Paystack response", nil)
		return
	}

	if !paystackResp.Status {
		utils.BuildErrorResponse(w, http.StatusBadRequest, "Paystack initialization failed: "+paystackResp.Message, nil)
		return
	}

	tx := Transaction{
		WalletID:    wallet.ID,
		Reference:   reference,
		Type:        TransactionDeposit,
		Amount:      req.Amount,
		Status:      TransactionPending,
		Description: "Wallet Deposit via Paystack",
	}

	if err := h.Repo.CreateTransaction(&tx); err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to register transaction", nil)
		return
	}

	utils.BuildSuccessResponse(w, http.StatusOK, "Deposit initialized", paystackResp.Data)
}

func (h *Handler) PaystackWebhook(w http.ResponseWriter, r *http.Request) {
	secret := h.Config.PaystackSecret
	signature := r.Header.Get("x-paystack-signature")

	logger.Info("Webhook received", logger.Fields{"remote_addr": r.RemoteAddr})

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Webhook: Failed to read body", logger.Fields{"error": err.Error()})
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hash := hmac.New(sha512.New, []byte(secret))
	hash.Write(body)
	expectedSig := hex.EncodeToString(hash.Sum(nil))

	if signature != expectedSig {
		logger.Error("Webhook: Signature mismatch", logger.Fields{
			"received": signature,
			"expected": expectedSig,
		})
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var event struct {
		Event string `json:"event"`
		Data  struct {
			Reference string `json:"reference"`
			Status    string `json:"status"`
			Amount    int64  `json:"amount"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &event); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if event.Event == "charge.success" {
		tx, err := h.Repo.GetTransactionByReference(event.Data.Reference)
		if err != nil {
			logger.Warn("Webhook: Transaction not found", logger.Fields{"reference": event.Data.Reference})
			w.WriteHeader(http.StatusOK)
			return
		}

		if tx.Status == TransactionSuccess {
			w.WriteHeader(http.StatusOK)
			return
		}

		if err := h.Repo.CreditWallet(tx.WalletID.String(), event.Data.Amount); err != nil {
			logger.Error("Webhook: Failed to credit wallet", logger.Fields{"error": err.Error(), "wallet_id": tx.WalletID.String()})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := h.Repo.UpdateTransactionStatus(event.Data.Reference, TransactionSuccess); err != nil {
			logger.Error("CRITICAL: Balance credited but status update failed", logger.Fields{"reference": event.Data.Reference, "error": err.Error()})
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		logger.Info("Webhook: Transaction processed successfully", logger.Fields{"reference": event.Data.Reference})
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetWalletBalance(w http.ResponseWriter, r *http.Request) {
	usr, _ := r.Context().Value(utils.UserKey).(user.User)

	wallet, err := h.Repo.GetWalletByUserID(usr.ID.String())
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusNotFound, "Wallet not found", nil)
		return
	}

	utils.BuildSuccessResponse(w, http.StatusOK, "Wallet Balance", map[string]any{
		"balance": wallet.Balance,
	})
}

func (h *Handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	usr, _ := r.Context().Value(utils.UserKey).(user.User)

	wallet, err := h.Repo.GetWalletByUserID(usr.ID.String())
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusNotFound, "Wallet not found", nil)
		return
	}

	utils.BuildSuccessResponse(w, http.StatusOK, "Wallet Details", wallet)
}

type TransferRequest struct {
	WalletNumber string `json:"wallet_number"`
	Amount       int64  `json:"amount"`
	Pin          string `json:"pin"`
	Description  string `json:"description"`
}

func (h *Handler) TransferFunds(w http.ResponseWriter, r *http.Request) {
	usr, _ := r.Context().Value(utils.UserKey).(user.User)

	var req TransferRequest
	if status, err := utils.DecodeJSONBody(w, r, &req); err != nil {
		utils.BuildErrorResponse(w, status, "Invalid request", map[string]string{"error": err.Error()})
		return
	}

	if req.Amount < h.Config.MinTransactionAmount {
		utils.BuildErrorResponse(w, http.StatusBadRequest, "Invalid amount, can't be less than 100 Naira (10000 Kobo)", nil)
		return
	}

	senderWallet, err := h.Repo.GetWalletByUserID(usr.ID.String())
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusNotFound, "Sender wallet not found", nil)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(senderWallet.PinHash), []byte(req.Pin)); err != nil {
		utils.BuildErrorResponse(w, http.StatusUnauthorized, "Invalid PIN", nil)
		return
	}

	if senderWallet.WalletNumber == req.WalletNumber {
		utils.BuildErrorResponse(w, http.StatusBadRequest, "Cannot transfer to self", nil)
		return
	}

	recipientWallet, err := h.Repo.GetWalletByNumber(req.WalletNumber)
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusNotFound, "Recipient wallet not found", nil)
		return
	}

	reference := fmt.Sprintf("trf-%d", time.Now().UnixNano())
	if err := h.Repo.TransferFunds(senderWallet.ID.String(), recipientWallet.ID.String(), senderWallet.WalletNumber, recipientWallet.WalletNumber, reference, req.Amount, req.Description); err != nil {
		if err.Error() == "insufficient balance" {
			utils.BuildErrorResponse(w, http.StatusBadRequest, "Insufficient balance", nil)
		} else {
			utils.BuildErrorResponse(w, http.StatusInternalServerError, "Transfer failed", map[string]string{"error": err.Error()})
		}
		return
	}

	utils.BuildSuccessResponse(w, http.StatusOK, "Transfer completed", nil)
}

func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	usr, _ := r.Context().Value(utils.UserKey).(user.User)

	wallet, err := h.Repo.GetWalletByUserID(usr.ID.String())
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusNotFound, "Wallet not found", nil)
		return
	}

	limit, offset, page := utils.GetPaginationDetails(r)

	txs, err := h.Repo.GetTransactions(wallet.ID.String(), limit, offset)
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to fetch transactions", nil)
		return
	}

	count, _ := h.Repo.CountTransactions(wallet.ID.String())
	totalPages := int(math.Ceil(float64(count) / float64(limit)))

	utils.BuildSuccessResponse(w, http.StatusOK, "Transaction History", map[string]interface{}{
		"transactions": txs,
		"meta": map[string]interface{}{
			"total_items":  count,
			"total_pages":  totalPages,
			"current_page": page,
			"limit":        limit,
		},
	})
}

func (h *Handler) GetDepositStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	reference := vars["reference"]
	if reference == "" || !strings.HasPrefix(reference, "dep-") {
		utils.BuildErrorResponse(w, http.StatusBadRequest, "Invalid reference format", nil)
		return
	}

	tx, err := h.Repo.GetTransactionByReference(reference)
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusNotFound, "Transaction not found", nil)
		return
	}

	response := map[string]interface{}{
		"reference": tx.Reference,
		"status":    tx.Status,
		"amount":    tx.Amount,
	}

	if tx.Status == TransactionPending {
		paystackStatus, err := h.verifyPaystackStatus(reference)
		if err == nil {
			response["paystack_status"] = paystackStatus
		} else {
			response["paystack_status"] = "unknown"
			response["paystack_error"] = err.Error()
		}
	} else {
		response["paystack_status"] = "not_checked"
	}

	utils.BuildSuccessResponse(w, http.StatusOK, "Transaction status retrieved", response)
}

func (h *Handler) verifyPaystackStatus(reference string) (string, error) {
	url := fmt.Sprintf("https://api.paystack.co/transaction/verify/%s", reference)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+h.Config.PaystackSecret)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("paystack returned status %d", resp.StatusCode)
	}

	var result struct {
		Status  bool   `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Status string `json:"status"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if !result.Status {
		return "", fmt.Errorf("paystack verification failed: %s", result.Message)
	}

	return result.Data.Status, nil
}

func generateWalletNumber() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("%010d", r.Int63n(10000000000))
}
