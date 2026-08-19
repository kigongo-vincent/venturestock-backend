package sale

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/middleware"
)

func RouterRegister(r *base.RouteRegister) {

	s := NewService(r.DB)
	g := r.Router.Group("sales", middleware.VerifyJWT())
	g.Get("/", s.FindAll)
	g.Post("/", s.Create)
	g.Put("/:id", s.Update)
	g.Get("/seller/:userId", s.FindBySeller)
}
