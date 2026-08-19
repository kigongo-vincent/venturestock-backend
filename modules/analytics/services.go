package analytics

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/product"
	"github.com/kigongo-vincent/venturestock-backend/modules/sale"
	"github.com/kigongo-vincent/venturestock-backend/modules/shared"
	"gorm.io/gorm"
)

func NewService(db *gorm.DB) *AnalyticsService {
	return &AnalyticsService{DB: db}
}

func (s *AnalyticsService) Home(c *fiber.Ctx) error {

	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	m, mErr := s.GetMinReport(UserID)
	if mErr != nil {
		return base.API_ERROR(c, "failed to get minireport")
	}

	w, wErr := s.GetWeekSales(UserID)
	if wErr != nil {
		return base.API_ERROR(c, "failed to get week sales")
	}

	p, pErr := s.GetMonthSales(UserID)
	if pErr != nil {
		return base.API_ERROR(c, "failed to get month profit")
	}

	b, bErr := s.GetBranchSales(UserID)
	if bErr != nil {
		return base.API_ERROR(c, "failed to get branch sales")
	}

	pr, prErr := s.GetProductSales(UserID)
	if prErr != nil {
		return base.API_ERROR(c, "failed to get product sales")
	}
	data := HomeAnalytics{
		Reports:      m,
		WeekSales:    w,
		MonthSales:   p,
		BranchSales:  b,
		ProductSales: pr,
	}
	return c.JSON(base.APIResponse{Data: data})

}

// getUserBranchIDs returns the branches a user's analytics should be
// scoped to:
//   - if the user OWNS a business, they see every branch under that
//     business (full business roll-up), regardless of their own
//     branch_members rows
//   - otherwise, they see only the branches they are an explicit
//     member of
//
// These two cases are intentionally NOT merged — an owner isn't
// additionally limited to their branch_members rows, and a plain
// member isn't given business-wide visibility just because someone
// else owns it.
func (s *AnalyticsService) getUserBranchIDs(userID uint) ([]uint, error) {

	var ownedBusinessIDs []uint
	if err := s.DB.Model(&shared.Business{}).
		Where("owner_id = ?", userID).
		Pluck("id", &ownedBusinessIDs).Error; err != nil {
		return nil, err
	}

	if len(ownedBusinessIDs) > 0 {
		var ownedBranches []product.Branch
		if err := s.DB.
			Where("business_id IN ?", ownedBusinessIDs).
			Find(&ownedBranches).Error; err != nil {
			return nil, err
		}
		ids := make([]uint, 0, len(ownedBranches))
		for _, b := range ownedBranches {
			ids = append(ids, b.ID)
		}
		return ids, nil
	}

	// not an owner — scope to explicit branch membership only
	var memberBranches []product.Branch
	if err := s.DB.
		Joins("JOIN branch_members ON branch_members.branch_id = branches.id").
		Where("branch_members.user_id = ?", userID).
		Find(&memberBranches).Error; err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(memberBranches))
	for _, b := range memberBranches {
		ids = append(ids, b.ID)
	}
	return ids, nil
}

// getUserProductIDs returns the IDs of all products belonging to the user's branches.
func (s *AnalyticsService) getUserProductIDs(branchIDs []uint) ([]uint, error) {
	if len(branchIDs) == 0 {
		return []uint{}, nil
	}

	var products []product.Product
	if err := s.DB.
		Where("branch_id IN ?", branchIDs).
		Find(&products).Error; err != nil {
		return nil, err
	}

	ids := make([]uint, 0, len(products))
	for _, p := range products {
		ids = append(ids, p.ID)
	}
	return ids, nil
}

// ... GetMinReport, GetWeekSales, GetProductSales, GetBranchSales,
// GetMonthSales, GetReportsSummary, Reports all remain exactly as
// they were — none of them need to change, since they all consume
// getUserBranchIDs as their single source of scope.

// GetMinReport returns aggregate stock count and total sales value for a user's branches.
func (s *AnalyticsService) GetMinReport(UserID uint) (MiniReport, error) {
	branchIDs, err := s.getUserBranchIDs(UserID)
	if err != nil {
		return MiniReport{}, err
	}

	var products []product.Product
	if len(branchIDs) > 0 {
		if err := s.DB.Where("branch_id IN ?", branchIDs).Find(&products).Error; err != nil {
			return MiniReport{}, err
		}
	}

	var totalStock uint
	for _, p := range products {
		totalStock += p.Quantity.Data().Value
	}

	productIDs := make([]uint, 0, len(products))
	for _, p := range products {
		productIDs = append(productIDs, p.ID)
	}

	var sales []sale.Sale
	if len(productIDs) > 0 {
		if err := s.DB.Where("product_id IN ?", productIDs).Find(&sales).Error; err != nil {
			return MiniReport{}, err
		}
	}

	var totalSales uint
	currency := product.UGX
	for _, sl := range sales {
		amt := sl.Amount.Data()
		totalSales += amt.Value
		currency = amt.Currency
	}

	return MiniReport{
		Stock: totalStock,
		Sales: product.Price{
			Currency: currency,
			Value:    totalSales,
		},
	}, nil
}

// GetWeekSales returns total sales per day for the last 7 days for a user's branches.
func (s *AnalyticsService) GetWeekSales(UserID uint) ([]WeekSale, error) {
	branchIDs, err := s.getUserBranchIDs(UserID)
	if err != nil {
		return nil, err
	}

	productIDs, err := s.getUserProductIDs(branchIDs)
	if err != nil {
		return nil, err
	}

	weekStart := time.Now().AddDate(0, 0, -6)
	dayTotals := map[string]uint{}
	dayOrder := make([]string, 0, 7)

	for i := 0; i < 7; i++ {
		d := weekStart.AddDate(0, 0, i)
		key := d.Format("Mon")
		dayOrder = append(dayOrder, key)
		dayTotals[key] = 0
	}

	if len(productIDs) > 0 {
		var sales []sale.Sale
		if err := s.DB.
			Where("product_id IN ? AND created_at >= ?", productIDs, weekStart).
			Find(&sales).Error; err != nil {
			return nil, err
		}

		for _, sl := range sales {
			key := sl.CreatedAt.Format("Mon")
			if _, ok := dayTotals[key]; ok {
				dayTotals[key] += sl.Amount.Data().Value
			}
		}
	}

	result := make([]WeekSale, 0, len(dayOrder))
	for _, day := range dayOrder {
		result = append(result, WeekSale{Day: day, Sales: dayTotals[day]})
	}
	return result, nil
}

// GetProductSales returns total sales value grouped by product for a user's branches.
func (s *AnalyticsService) GetProductSales(UserID uint) ([]ProductSale, error) {
	branchIDs, err := s.getUserBranchIDs(UserID)
	if err != nil {
		return nil, err
	}

	var products []product.Product
	if len(branchIDs) > 0 {
		if err := s.DB.Where("branch_id IN ?", branchIDs).Find(&products).Error; err != nil {
			return nil, err
		}
	}

	result := make([]ProductSale, 0, len(products))
	for _, p := range products {
		var sales []sale.Sale
		if err := s.DB.Where("product_id = ?", p.ID).Find(&sales).Error; err != nil {
			return nil, err
		}

		var total uint
		for _, sl := range sales {
			total += sl.Amount.Data().Value
		}

		result = append(result, ProductSale{Product: p.Name, Sales: total})
	}
	return result, nil
}

// GetBranchSales returns total sales value grouped by branch for a user's branches.
func (s *AnalyticsService) GetBranchSales(UserID uint) ([]BranchSale, error) {
	branchIDs, err := s.getUserBranchIDs(UserID)
	if err != nil {
		return nil, err
	}

	var branches []product.Branch
	if len(branchIDs) > 0 {
		if err := s.DB.Where("id IN ?", branchIDs).Find(&branches).Error; err != nil {
			return nil, err
		}
	}

	result := make([]BranchSale, 0, len(branches))
	for _, b := range branches {
		var products []product.Product
		if err := s.DB.Where("branch_id = ?", b.ID).Find(&products).Error; err != nil {
			return nil, err
		}

		productIDs := make([]uint, 0, len(products))
		for _, p := range products {
			productIDs = append(productIDs, p.ID)
		}

		var total uint
		if len(productIDs) > 0 {
			var sales []sale.Sale
			if err := s.DB.Where("product_id IN ?", productIDs).Find(&sales).Error; err != nil {
				return nil, err
			}
			for _, sl := range sales {
				total += sl.Amount.Data().Value
			}
		}

		result = append(result, BranchSale{Branch: b.Name, Sales: total})
	}
	return result, nil
}
func (s *AnalyticsService) GetMonthSales(UserID uint) ([]MonthSales, error) {
	branchIDs, err := s.getUserBranchIDs(UserID)
	if err != nil {
		return nil, err
	}

	productIDs, err := s.getUserProductIDs(branchIDs)
	if err != nil {
		return nil, err
	}

	monthTotals := map[string]uint{}
	monthOrder := make([]string, 0, 12)
	now := time.Now()

	for i := 8; i >= 0; i-- {
		m := now.AddDate(0, -i, 0)
		key := m.Format("Jan")
		monthOrder = append(monthOrder, key)
		monthTotals[key] = 0
	}

	if len(productIDs) > 0 {
		startDate := now.AddDate(0, -11, 0)
		var sales []sale.Sale
		if err := s.DB.
			Where("product_id IN ? AND created_at >= ?", productIDs, startDate).
			Find(&sales).Error; err != nil {
			return nil, err
		}

		for _, sl := range sales {
			key := sl.CreatedAt.Format("Jan")
			if _, ok := monthTotals[key]; ok {
				monthTotals[key] += sl.Amount.Data().Value
			}
		}
	}

	result := make([]MonthSales, 0, len(monthOrder))
	for _, month := range monthOrder {
		result = append(result, MonthSales{Month: month, Sales: monthTotals[month]})
	}
	return result, nil
}

func (s *AnalyticsService) GetReportsSummary(UserID uint) (ReportsSummary, error) {
	branchIDs, err := s.getUserBranchIDs(UserID)
	if err != nil {
		return ReportsSummary{}, err
	}

	var products []product.Product
	if len(branchIDs) > 0 {
		if err := s.DB.Where("branch_id IN ?", branchIDs).Find(&products).Error; err != nil {
			return ReportsSummary{}, err
		}
	}

	var totalStock uint
	var totalStockValue uint
	var lowStockAlerts uint
	currency := product.UGX

	for _, p := range products {
		qty := p.Quantity.Data().Value
		price := p.Price.Data()

		totalStock += qty
		totalStockValue += qty * price.Value
		currency = price.Currency

		// low stock threshold: adjust to whatever your business actually
		// considers "low" — arbitrarily using <5 units since Product has
		// an Availability field but it's gorm:"-" (not persisted) and unused
		if qty < 5 {
			lowStockAlerts++
		}
	}

	productIDs := make([]uint, 0, len(products))
	for _, p := range products {
		productIDs = append(productIDs, p.ID)
	}

	var salesToday uint
	var salesThisMonth uint

	if len(productIDs) > 0 {
		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

		var sales []sale.Sale
		if err := s.DB.
			Where("product_id IN ? AND created_at >= ?", productIDs, startOfMonth).
			Find(&sales).Error; err != nil {
			return ReportsSummary{}, err
		}

		for _, sl := range sales {
			amt := sl.Amount.Data()
			salesThisMonth += amt.Value
			if sl.CreatedAt.After(startOfDay) || sl.CreatedAt.Equal(startOfDay) {
				salesToday += amt.Value
			}
			currency = amt.Currency
		}
	}

	return ReportsSummary{
		TotalStockValue: product.Price{Currency: currency, Value: totalStockValue},
		TotalStock:      totalStock,
		SalesToday:      product.Price{Currency: currency, Value: salesToday},
		SalesThisMonth:  product.Price{Currency: currency, Value: salesThisMonth},
		LowStockAlerts:  lowStockAlerts,
	}, nil
}

func (s *AnalyticsService) Reports(c *fiber.Ctx) error {
	UserID, uErr := base.GetUserIDFromCtx(c)
	if uErr != nil {
		return base.API_ERROR(c, "user not found")
	}

	summary, err := s.GetReportsSummary(UserID)
	if err != nil {
		return base.API_ERROR(c, "failed to get reports summary")
	}

	return c.JSON(base.APIResponse{Data: summary})
}
