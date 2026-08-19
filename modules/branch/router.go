package branch

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/middleware"
)

func RouterRegister(r *base.RouteRegister) {

	s := NewService(r.DB)
	g := r.Router.Group("branches", middleware.VerifyJWT())
	g.Get("/", s.FindAll)
	g.Get("/:id", s.FindOne)
	g.Get("/:id/reports", s.Reports)
	g.Get("/:id/users", s.FindUsers)
	g.Post("/", s.CreateHandler)

}
