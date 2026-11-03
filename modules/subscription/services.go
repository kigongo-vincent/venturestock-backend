package subscription

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrNotBusinessOwner = errors.New("only the business owner can manage the subscription")

type Service struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

func SeedPackages(db *gorm.DB) error {
	defaults := []Package{
		{
			Slug:            FreePackageSlug,
			Name:            "Free",
			Description:     "Three months of VentureStock at no cost.",
			Amount:          0,
			Currency:        "USD",
			BillingInterval: "month",
			IntervalCount:   3,
			Features:        datatypes.NewJSONType([]string{"3 months free", "1 branch", "Up to 3 workers"}),
			SortOrder:       1,
			IsActive:        true,
		},
		{
			Slug:            ProPackageSlug,
			Name:            "Pro",
			Description:     "Unlimited access billed by the hour.",
			Amount:          10,
			Currency:        "USD",
			BillingInterval: "hour",
			IntervalCount:   1,
			Features:        datatypes.NewJSONType([]string{"Unlimited branches", "Unlimited workers", "Priority support"}),
			SortOrder:       2,
			IsActive:        true,
		},
	}

	for i := range defaults {
		var existing Package
		err := db.Where("slug = ?", defaults[i].Slug).First(&existing).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := db.Create(&defaults[i]).Error; err != nil {
			return err
		}
	}

	return nil
}

func BackfillFreeSubscriptions(db *gorm.DB) error {
	pkg, err := NewService(db).packageBySlug(FreePackageSlug)
	if err != nil {
		return err
	}

	var businessIDs []uint
	if err := db.Model(&shared.Business{}).Pluck("id", &businessIDs).Error; err != nil {
		return err
	}

	now := time.Now()
	for _, businessID := range businessIDs {
		var count int64
		if err := db.Model(&Subscription{}).Where("business_id = ?", businessID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := db.Create(&Subscription{
			BusinessID: businessID,
			PackageID:  pkg.ID,
			Status:     StatusActive,
			StartsAt:   now,
			ExpiresAt:  pkg.ExpiresFrom(now),
		}).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) CreateFreeSubscription(businessID uint) error {
	pkg, err := s.packageBySlug(FreePackageSlug)
	if err != nil {
		return err
	}

	var existing Subscription
	err = s.DB.Where("business_id = ?", businessID).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	now := time.Now()
	return s.DB.Create(&Subscription{
		BusinessID: businessID,
		PackageID:  pkg.ID,
		Status:     StatusActive,
		StartsAt:   now,
		ExpiresAt:  pkg.ExpiresFrom(now),
	}).Error
}

func (s *Service) AttachSnapshot(business *shared.Business) {
	if business == nil {
		return
	}
	business.Subscription = s.SnapshotForBusiness(business.ID)
}

func (s *Service) SnapshotForBusiness(businessID uint) *shared.SubscriptionSnapshot {
	var sub Subscription
	err := s.DB.Preload("Package").Where("business_id = ?", businessID).First(&sub).Error
	if err != nil {
		return &shared.SubscriptionSnapshot{Status: "missing", IsActive: false}
	}

	status := sub.Status
	active := status == StatusActive && sub.ExpiresAt.After(time.Now())
	if status == StatusActive && !active {
		status = StatusExpired
		s.DB.Model(&Subscription{}).Where("id = ?", sub.ID).Update("status", StatusExpired)
	}

	snapshot := &shared.SubscriptionSnapshot{
		ID:        sub.ID,
		Status:    status,
		IsActive:  active,
		StartsAt:  sub.StartsAt,
		ExpiresAt: sub.ExpiresAt,
	}
	if sub.Package != nil {
		snapshot.PackageSlug = sub.Package.Slug
		snapshot.PackageName = sub.Package.Name
	}
	return snapshot
}

func (s *Service) Packages(c *fiber.Ctx) error {
	var packages []Package
	if err := s.DB.Where("is_active = ?", true).Order("sort_order asc, amount asc").Find(&packages).Error; err != nil {
		return base.API_ERROR(c, "failed to load subscription packages")
	}
	return c.JSON(base.APIResponse{Data: packages})
}

func (s *Service) Current(c *fiber.Ctx) error {
	business, err := s.currentOwnerBusiness(c)
	if err != nil {
		return subscriptionError(c, err)
	}

	return s.overviewResponse(c, business.ID)
}

func (s *Service) Checkout(c *fiber.Ctx) error {
	business, err := s.currentOwnerBusiness(c)
	if err != nil {
		return subscriptionError(c, err)
	}

	var body CheckoutRequest
	if err := c.BodyParser(&body); err != nil {
		return base.API_ERROR(c, "invalid subscription package")
	}

	var pkg Package
	if err := s.DB.Where("id = ? AND is_active = ?", body.PackageID, true).First(&pkg).Error; err != nil {
		return base.API_ERROR(c, "subscription package not found")
	}
	if pkg.Amount <= 0 {
		return base.API_ERROR(c, "the free package does not require payment")
	}

	var owner shared.User
	if business.OwnerID == nil || s.DB.First(&owner, *business.OwnerID).Error != nil {
		return base.API_ERROR(c, "business owner account not found")
	}

	client, err := NewFlutterwaveClient()
	if errors.Is(err, ErrFlutterwaveNotConfigured) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(base.APIResponse{Message: "payment provider is not configured"})
	}
	if err != nil {
		return base.API_ERROR(c, "failed to initialize payment provider")
	}

	txRef, err := newTransactionReference(business.ID)
	if err != nil {
		return base.API_ERROR(c, "failed to create payment reference")
	}

	payment := Payment{
		BusinessID: business.ID,
		PackageID:  pkg.ID,
		TxRef:      txRef,
		Amount:     pkg.Amount,
		Currency:   pkg.Currency,
		Status:     StatusPending,
	}
	if err := s.DB.Create(&payment).Error; err != nil {
		return base.API_ERROR(c, "failed to create payment record")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	redirectURL := strings.TrimSpace(os.Getenv("FLW_REDIRECT_URL"))
	if redirectURL == "" {
		redirectURL = "venturestock://settings/subscription"
	}

	link, raw, err := client.Initialize(ctx, InitializePaymentRequest{
		TxRef:       txRef,
		Amount:      pkg.Amount,
		Currency:    pkg.Currency,
		RedirectURL: redirectURL,
		Customer: FlutterwaveCustomer{
			Name:        owner.Name,
			Email:       owner.Email,
			PhoneNumber: owner.Phone,
		},
		Customizations: map[string]string{
			"title":       "VentureStock Subscription",
			"description": fmt.Sprintf("%s plan", pkg.Name),
		},
		Meta: map[string]string{
			"businessId": fmt.Sprintf("%d", business.ID),
			"packageId":  fmt.Sprintf("%d", pkg.ID),
			"txRef":      txRef,
		},
	})
	if err != nil {
		s.DB.Model(&payment).Updates(map[string]interface{}{
			"status":       StatusFailed,
			"raw_response": raw,
		})
		return c.Status(fiber.StatusBadGateway).JSON(base.APIResponse{Message: err.Error()})
	}

	payment.CheckoutURL = link
	payment.RawResponse = raw
	if err := s.DB.Save(&payment).Error; err != nil {
		return base.API_ERROR(c, "failed to save payment details")
	}

	return c.JSON(base.APIResponse{Data: CheckoutResponse{
		CheckoutURL: link,
		TxRef:       txRef,
		PaymentID:   payment.ID,
	}})
}

func (s *Service) Verify(c *fiber.Ctx) error {
	business, err := s.currentOwnerBusiness(c)
	if err != nil {
		return subscriptionError(c, err)
	}

	var body VerifyRequest
	if err := c.BodyParser(&body); err != nil || strings.TrimSpace(body.TxRef) == "" {
		return base.API_ERROR(c, "payment reference is required")
	}

	var payment Payment
	if err := s.DB.Where("tx_ref = ? AND business_id = ?", body.TxRef, business.ID).First(&payment).Error; err != nil {
		return base.API_ERROR(c, "payment not found")
	}

	if payment.Status != StatusSuccess {
		client, clientErr := NewFlutterwaveClient()
		if clientErr != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(base.APIResponse{Message: "payment provider is not configured"})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		tx, raw, verifyErr := client.VerifyByReference(ctx, payment.TxRef)
		if verifyErr != nil {
			return c.Status(fiber.StatusBadGateway).JSON(base.APIResponse{Message: verifyErr.Error()})
		}

		if tx.Status != "successful" {
			s.DB.Model(&payment).Updates(map[string]interface{}{
				"status":       StatusFailed,
				"raw_response": raw,
			})
			return base.API_ERROR(c, "payment has not been completed")
		}

		if err := validateTransaction(&payment, tx); err != nil {
			return base.API_ERROR(c, err.Error())
		}

		if err := s.activatePayment(&payment, tx, raw); err != nil {
			return base.API_ERROR(c, "failed to activate subscription")
		}
	}

	return s.overviewResponse(c, business.ID)
}

func (s *Service) Webhook(c *fiber.Ctx) error {
	if !validWebhookSignature(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(base.APIResponse{Message: "invalid webhook signature"})
	}

	var payload WebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(base.APIResponse{Message: "invalid webhook payload"})
	}
	if payload.Data.TxRef == "" {
		return c.SendStatus(fiber.StatusOK)
	}

	var payment Payment
	if err := s.DB.Where("tx_ref = ?", payload.Data.TxRef).First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.SendStatus(fiber.StatusOK)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(base.APIResponse{Message: "failed to load payment"})
	}

	if payment.Status == StatusSuccess {
		return c.SendStatus(fiber.StatusOK)
	}

	if payload.Data.Status != "" && payload.Data.Status != "successful" {
		s.DB.Model(&payment).Update("status", StatusFailed)
		return c.SendStatus(fiber.StatusOK)
	}

	client, err := NewFlutterwaveClient()
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(base.APIResponse{Message: "payment provider is not configured"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	tx, raw, err := client.VerifyByReference(ctx, payment.TxRef)
	if err != nil {
		log.Printf("subscription webhook verification failed for %s: %v", payment.TxRef, err)
		return c.Status(fiber.StatusBadGateway).JSON(base.APIResponse{Message: "failed to verify payment"})
	}
	if tx.Status != "successful" {
		s.DB.Model(&payment).Updates(map[string]interface{}{
			"status":       StatusFailed,
			"raw_response": raw,
		})
		return c.SendStatus(fiber.StatusOK)
	}
	if err := validateTransaction(&payment, tx); err != nil {
		log.Printf("subscription webhook validation failed for %s: %v", payment.TxRef, err)
		return c.Status(fiber.StatusBadRequest).JSON(base.APIResponse{Message: err.Error()})
	}
	if err := s.activatePayment(&payment, tx, raw); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(base.APIResponse{Message: "failed to activate subscription"})
	}

	return c.SendStatus(fiber.StatusOK)
}

func (s *Service) overviewResponse(c *fiber.Ctx, businessID uint) error {
	var packages []Package
	if err := s.DB.Where("is_active = ?", true).Order("sort_order asc, amount asc").Find(&packages).Error; err != nil {
		return base.API_ERROR(c, "failed to load subscription packages")
	}

	var payments []Payment
	if err := s.DB.Preload("Package").
		Where("business_id = ?", businessID).
		Order("created_at desc").
		Limit(20).
		Find(&payments).Error; err != nil {
		return base.API_ERROR(c, "failed to load payment history")
	}

	return c.JSON(base.APIResponse{Data: Overview{
		Subscription: s.SnapshotForBusiness(businessID),
		Packages:     packages,
		Payments:     payments,
	}})
}

func (s *Service) activatePayment(payment *Payment, tx *FlutterwaveTransaction, raw string) error {
	return s.DB.Transaction(func(db *gorm.DB) error {
		var current Payment
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, payment.ID).Error; err != nil {
			return err
		}
		if current.Status == StatusSuccess {
			*payment = current
			return nil
		}

		var pkg Package
		if err := db.First(&pkg, current.PackageID).Error; err != nil {
			return err
		}

		now := time.Now()
		baseTime := now

		var existing Subscription
		err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("business_id = ?", current.BusinessID).
			First(&existing).Error
		switch {
		case err == nil:
			if existing.Status == StatusActive && existing.ExpiresAt.After(now) && existing.PackageID == pkg.ID {
				baseTime = existing.ExpiresAt
			}
			existing.PackageID = pkg.ID
			existing.Status = StatusActive
			existing.StartsAt = now
			existing.ExpiresAt = pkg.ExpiresFrom(baseTime)
			existing.AutoRenew = false
			if err := db.Save(&existing).Error; err != nil {
				return err
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := db.Create(&Subscription{
				BusinessID: current.BusinessID,
				PackageID:  pkg.ID,
				Status:     StatusActive,
				StartsAt:   now,
				ExpiresAt:  pkg.ExpiresFrom(baseTime),
			}).Error; err != nil {
				return err
			}
		default:
			return err
		}

		paidAt := now
		current.Status = StatusSuccess
		current.FlutterwaveID = &tx.ID
		current.PaidAt = &paidAt
		current.RawResponse = raw
		if err := db.Save(&current).Error; err != nil {
			return err
		}

		*payment = current
		return nil
	})
}

func (s *Service) currentOwnerBusiness(c *fiber.Ctx) (*shared.Business, error) {
	userID, err := base.GetUserIDFromCtx(c)
	if err != nil {
		return nil, err
	}

	var business shared.Business
	if err := s.DB.Where("owner_id = ?", userID).First(&business).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotBusinessOwner
		}
		return nil, err
	}
	return &business, nil
}

func (s *Service) packageBySlug(slug string) (Package, error) {
	var pkg Package
	if err := s.DB.Where("slug = ? AND is_active = ?", slug, true).First(&pkg).Error; err != nil {
		return Package{}, err
	}
	return pkg, nil
}

func validateTransaction(payment *Payment, tx *FlutterwaveTransaction) error {
	if tx.TxRef != "" && tx.TxRef != payment.TxRef {
		return errors.New("payment reference does not match")
	}
	if !strings.EqualFold(tx.Currency, payment.Currency) {
		return errors.New("payment currency does not match")
	}
	if tx.Amount+0.005 < payment.Amount {
		return errors.New("payment amount is less than expected")
	}
	return nil
}

func validWebhookSignature(c *fiber.Ctx) bool {
	secret := strings.TrimSpace(os.Getenv("FLW_SECRET_HASH"))
	if secret == "" {
		return false
	}

	providedHash := strings.TrimSpace(c.Get("verif-hash"))
	if providedHash != "" && subtle.ConstantTimeCompare([]byte(providedHash), []byte(secret)) == 1 {
		return true
	}

	providedSignature := strings.TrimSpace(c.Get("flutterwave-signature"))
	if providedSignature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(c.Body())
	sum := mac.Sum(nil)

	if hmac.Equal([]byte(strings.ToLower(providedSignature)), []byte(hex.EncodeToString(sum))) {
		return true
	}
	return hmac.Equal([]byte(providedSignature), []byte(base64.StdEncoding.EncodeToString(sum)))
}

func newTransactionReference(businessID uint) (string, error) {
	random := make([]byte, 6)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return fmt.Sprintf("vs_%d_%d_%s", businessID, time.Now().Unix(), hex.EncodeToString(random)), nil
}

func subscriptionError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrNotBusinessOwner):
		return c.Status(fiber.StatusForbidden).JSON(base.APIResponse{Message: ErrNotBusinessOwner.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		return c.Status(fiber.StatusForbidden).JSON(base.APIResponse{Message: "you do not have access to this subscription"})
	default:
		return base.API_ERROR(c, "failed to load subscription")
	}
}
