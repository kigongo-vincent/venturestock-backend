package branch

import "gorm.io/gorm"

func NewService(db *gorm.DB) *BranchService {
	return &BranchService{DB: db}
}

func (s *BranchService) Create(branch Branch) (uint, error) {
	if err := s.DB.Create(&branch).Error; err != nil {
		return 0, err
	}
	return branch.ID, nil
}

func (s *BranchService) AddMember(BranchID, UserID uint) error {
	return s.DB.Table("branch_members").Create(map[string]interface{}{"user_id": UserID, "branch_id": BranchID}).Error
}
