package business

import "gorm.io/gorm"

func NewService(db *gorm.DB) *BusinessService {
	return &BusinessService{DB: db}
}

func (s *BusinessService) Create(business Business) (uint, error) {

	err := s.DB.Create(&business).Error
	if err != nil {
		return 0, err
	}

	return business.ID, nil

}
