package subscription

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/middleware"
)

func RouterRegister(r *base.RouteRegister) {
	s := NewService(r.DB)

	r.Router.Post("subscription/webhook", s.Webhook)

	g := r.Router.Group("subscription", middleware.VerifyJWT())
	g.Get("/", s.Current)
	g.Get("/packages", s.Packages)
	g.Post("/checkout", s.Checkout)
	g.Post("/verify", s.Verify)
}
