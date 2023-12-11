package morondanga

import "github.com/labstack/echo/v4"

// Group creates a new router group with prefix and optional group-level middleware.
func (s *Service) Group(name string) *echo.Group {
	return s.server.Group(name)
}

// GET registers a new GET route for a path with matching handler in the router
// with optional route-level middleware.
func (s *Service) GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.server.GET(path, h, m...)
}

// POST registers a new POST route for a path with matching handler in the router
// with optional route-level middleware.
func (s *Service) POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.server.POST(path, h, m...)
}

// PUT registers a new PUT route for a path with matching handler in the router
// with optional route-level middleware.
func (s *Service) PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.server.PUT(path, h, m...)
}

// PATCH registers a new PATCH route for a path with matching handler in the router
// with optional route-level middleware.
func (s *Service) PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.server.PATCH(path, h, m...)
}

// DELETE registers a new DELETE route for a path with matching handler in the router
// with optional route-level middleware.
func (s *Service) DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.server.DELETE(path, h, m...)
}

// HEAD registers a new HEAD route for a path with matching handler in the router
// with optional route-level middleware.
func (s *Service) HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.server.HEAD(path, h, m...)
}

// OPTIONS registers a new OPTIONS route for a path with matching handler in the router
// with optional route-level middleware.
func (s *Service) OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route {
	return s.server.OPTIONS(path, h, m...)
}
