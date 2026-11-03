package analytics

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/middleware"
)

func RouterRegister(r *base.RouteRegister) {

	s := NewService(r.DB)
	g := r.Router.Group("analytics", middleware.VerifyJWT(r.DB))

	g.Get("/home", s.Home)
	g.Get("/reports", s.Reports)

}
