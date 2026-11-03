package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var ErrFlutterwaveNotConfigured = errors.New("flutterwave is not configured")

type FlutterwaveClient struct {
	secretKey string
	baseURL   string
	client    *http.Client
}

type FlutterwaveCustomer struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

type InitializePaymentRequest struct {
	TxRef          string              `json:"tx_ref"`
	Amount         float64             `json:"amount"`
	Currency       string              `json:"currency"`
	RedirectURL    string              `json:"redirect_url"`
	Customer       FlutterwaveCustomer `json:"customer"`
	Customizations map[string]string   `json:"customizations,omitempty"`
	Meta           map[string]string   `json:"meta,omitempty"`
}

type FlutterwaveTransaction struct {
	ID       int64   `json:"id"`
	TxRef    string  `json:"tx_ref"`
	Status   string  `json:"status"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type initializeResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    struct {
		Link string `json:"link"`
	} `json:"data"`
}

type verifyResponse struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    FlutterwaveTransaction `json:"data"`
}

func NewFlutterwaveClient() (*FlutterwaveClient, error) {
	secretKey := strings.TrimSpace(os.Getenv("FLW_SECRET_KEY"))
	if secretKey == "" {
		return nil, ErrFlutterwaveNotConfigured
	}

	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("FLW_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = "https://api.flutterwave.com/v3"
	}

	return &FlutterwaveClient{
		secretKey: secretKey,
		baseURL:   baseURL,
		client:    &http.Client{Timeout: 20 * time.Second},
	}, nil
}

func (c *FlutterwaveClient) Initialize(ctx context.Context, payload InitializePaymentRequest) (string, string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/payments", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	raw, readErr := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if readErr != nil {
		return "", "", readErr
	}

	var parsed initializeResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", string(raw), fmt.Errorf("flutterwave returned an invalid response")
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 || parsed.Status != "success" || parsed.Data.Link == "" {
		message := parsed.Message
		if message == "" {
			message = "failed to initialize flutterwave payment"
		}
		return "", string(raw), errors.New(message)
	}

	return parsed.Data.Link, string(raw), nil
}

func (c *FlutterwaveClient) VerifyByReference(ctx context.Context, txRef string) (*FlutterwaveTransaction, string, error) {
	endpoint := c.baseURL + "/transactions/verify_by_reference?tx_ref=" + url.QueryEscape(txRef)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	raw, readErr := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if readErr != nil {
		return nil, "", readErr
	}

	var parsed verifyResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, string(raw), fmt.Errorf("flutterwave returned an invalid response")
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 || parsed.Status != "success" {
		message := parsed.Message
		if message == "" {
			message = "failed to verify flutterwave payment"
		}
		return nil, string(raw), errors.New(message)
	}

	return &parsed.Data, string(raw), nil
}
