package shared

import (
	"gorm.io/gorm"
)

func AddMember(DB *gorm.DB, BranchID, UserID uint) error {
	return DB.Table("branch_members").Create(map[string]interface{}{"user_id": UserID, "branch_id": BranchID}).Error
}

// GetUserBranchIDs returns the branches a user should see:
//   - business owners get every branch under businesses they own
//   - everyone else (workers) gets only branches they are a member of
func GetUserBranchIDs(DB *gorm.DB, userID uint) ([]uint, error) {
	var ownedBusinessIDs []uint
	if err := DB.Model(&Business{}).
		Where("owner_id = ?", userID).
		Pluck("id", &ownedBusinessIDs).Error; err != nil {
		return nil, err
	}

	if len(ownedBusinessIDs) > 0 {
		var ids []uint
		if err := DB.Model(&Branch{}).
			Where("business_id IN ?", ownedBusinessIDs).
			Pluck("id", &ids).Error; err != nil {
			return nil, err
		}
		return ids, nil
	}

	var ids []uint
	if err := DB.Table("branches").
		Joins("JOIN branch_members ON branch_members.branch_id = branches.id").
		Where("branch_members.user_id = ?", userID).
		Where("branches.deleted_at IS NULL").
		Pluck("branches.id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}
