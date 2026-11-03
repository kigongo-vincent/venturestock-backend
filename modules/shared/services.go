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

// UserHasBranchAccess reports whether a branch falls inside the user's
// visible scope (owned-business branches for owners, memberships for
// workers).
func UserHasBranchAccess(DB *gorm.DB, userID, branchID uint) (bool, error) {
	if branchID == 0 {
		return false, nil
	}

	branchIDs, err := GetUserBranchIDs(DB, userID)
	if err != nil {
		return false, err
	}
	return containsID(branchIDs, branchID), nil
}

// UsersShareBranch reports whether two users have at least one branch in
// common, using the requesting user's visible branch scope so owners also
// match workers in branches under their business.
func UsersShareBranch(DB *gorm.DB, requesterID, targetID uint) (bool, error) {
	if requesterID == targetID {
		return true, nil
	}

	branchIDs, err := GetUserBranchIDs(DB, requesterID)
	if err != nil {
		return false, err
	}
	if len(branchIDs) == 0 {
		return false, nil
	}

	var count int64
	if err := DB.Table("branch_members").
		Where("user_id = ? AND branch_id IN ?", targetID, branchIDs).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func containsID(ids []uint, id uint) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
