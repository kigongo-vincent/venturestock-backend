package auth

import "github.com/kigongo-vincent/venturestock-backend/modules/base"

func RouterRegister(r *base.RouteRegister) {

	s := NewService(r.DB)
	g := r.Router.Group("auth")

	g.Post("/login", s.Login)
	g.Post("/signup", s.Signup)
	g.Post("/googleAuth", s.GoogleAuth)

}
