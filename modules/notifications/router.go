package notifications

import (
	"github.com/kigongo-vincent/venturestock-backend/modules/base"
	"github.com/kigongo-vincent/venturestock-backend/modules/middleware"
)

func RouterRegister(r *base.RouteRegister) {

	s := NewService(r.DB)
	g := r.Router.Group("notifications", middleware.VerifyJWT())
	g.Get("/", s.FindAll)
	g.Get("/mark/:id", s.MarkOne)
	g.Get("/mark", s.MarkAll)
	g.Get("/unread-count", s.CountUnread)
}
