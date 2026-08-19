package test

import (
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/kigongo-vincent/venturestock-backend/modules/analytics"
)

func TestHomeAnalytics(t *testing.T) {
	HTTP[analytics.HomeAnalytics, interface{}](HTTPRequest[analytics.HomeAnalytics, interface{}]{Method: fiber.MethodGet, Path: "analytics/home"})
	t.Error()
}
