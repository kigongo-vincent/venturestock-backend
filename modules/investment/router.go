package investment

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/middleware"
)

func RouterRegister(r *base.RouteRegister) {
	s := NewInvestmentService(r.DB)

	g := r.Router.Group("/investments", middleware.VerifyJWT(r.DB))

	g.Get("/", s.FindAll)
	g.Get("/:id/comparison", s.Comparison)
	g.Post("/", s.Create)
	g.Delete("/:id", s.Delete)
}
