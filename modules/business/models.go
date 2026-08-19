package business

import (
	"gorm.io/gorm"
)

type BusinessService struct {
	DB *gorm.DB
}

type UpdateBusinessRequest struct {
	gorm.Model
	Name     string `json:"name"`
	Location string `json:"location"`
}
