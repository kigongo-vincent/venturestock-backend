package product

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/middleware"
)

func RouterRegister(r *base.RouteRegister) {

	s := NewService(r.DB)
	g := r.Router.Group("products", middleware.VerifyJWT())
	g.Get("/", s.FindAll)
	g.Post("/", s.Create)
	g.Put("/:id", s.Update)
	g.Put("/:id/stock", s.AdjustStock)

}
