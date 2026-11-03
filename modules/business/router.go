package business

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/middleware"
)

func RouterRegister(r *base.RouteRegister) {

	g := r.Router.Group("business", middleware.VerifyJWT(r.DB))
	s := NewService(r.DB)

	g.Put("/", s.Update)

}
