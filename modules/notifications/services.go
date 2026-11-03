package notifications

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"gorm.io/gorm"
)

func NewService(db *gorm.DB) *NotificationService {
	return &NotificationService{DB: db}
}

func (s *NotificationService) Create(notification Notification) error {
	return s.DB.Create(&notification).Error
}

func (s *NotificationService) FindAll(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	var notifications []Notification
	if err := s.DB.Where("user_id=?", UserID).Find(&notifications).Error; err != nil {
		return base.API_ERROR(c, "failed to get notifications")
	}
	return c.JSON(base.APIResponse{Data: notifications})

}

func (s *NotificationService) Mark(ID uint) error {
	var notification Notification
	if err := s.DB.First(&notification, uint(ID)).Error; err != nil {
		return err
	}

	notification.Read = true
	if err2 := s.DB.Save(&notification).Error; err2 != nil {
		return err2
	}

	return nil
}

func (s *NotificationService) MarkOne(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	param := c.Params("id")
	if param == "" {
		return base.API_ERROR(c, "failed to get param")
	}

	parsedParam, pErr := strconv.ParseUint(param, 10, 64)
	if pErr != nil {
		return base.API_ERROR(c, "failed to get param")
	}

	NotificationID := uint(parsedParam)
	var notification Notification
	if err := s.DB.Where("id = ? AND user_id = ?", NotificationID, UserID).First(&notification).Error; err != nil {
		return base.API_ERROR(c, "notification not found")
	}

	if err := s.Mark(NotificationID); err != nil {
		return base.API_ERROR(c, "failed to mark read")
	}

	return c.SendStatus(200)

}

func (s *NotificationService) MarkAll(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	var notifications []Notification
	if err := s.DB.Where("user_id=?", UserID).Find(&notifications).Error; err != nil {
		return base.API_ERROR(c, "failed to get notifications")
	}

	for _, n := range notifications {
		if err := s.Mark(n.ID); err != nil {
			return base.API_ERROR(c, "failed to mark read")
		}
	}

	return c.SendStatus(200)

}

func (s *NotificationService) CountUnread(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	var count int64
	if err := s.DB.Model(&Notification{}).
		Where("user_id = ? AND read = ?", UserID, false).
		Count(&count).Error; err != nil {
		return base.API_ERROR(c, "failed to count notifications")
	}

	return c.JSON(base.APIResponse{Data: fiber.Map{"count": count}})

}
