package user

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/middleware"
)

func RouterRegister(r *base.RouteRegister) {
	s := NewService(r.DB)
	g := r.Router.Group("users", middleware.VerifyJWT(r.DB))
	g.Post("/", s.CreateWorker)
	g.Put("/", s.UpdateAccount)
	g.Get("/:id", s.FindByID)
	g.Post("/otp", s.RequestOTP)
	g.Put("/email", s.UpdateEmail)
	g.Put("/phone", s.UpdatePhoneNumber)
}
