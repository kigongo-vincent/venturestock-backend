package expense

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/middleware"
)

func RouterRegister(r *base.RouteRegister) {

	s := NewService(r.DB)
	g := r.Router.Group("expenses", middleware.VerifyJWT())
	g.Post("/", s.Create)
	g.Get("/user/:userId", s.FindByUser)
	g.Get("/branch/:branchId", s.FindByBranch)
	g.Delete("/:id", s.Delete)

}
