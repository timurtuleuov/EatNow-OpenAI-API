package model

type GoogleVerifyRequest struct {
	Email         string `json:"email"`
	ProductID     string `json:"product_id"`
	PurchaseToken string `json:"purchase_token"`
	TransactionID string `json:"transaction_id"`
	Platform      string `json:"platform"`
}
