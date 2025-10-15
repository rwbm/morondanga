package morondanga

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/rwbm/morondanga/common"
	"github.com/rwbm/morondanga/middleware"
	"go.uber.org/zap"
)

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

// JWT returns the default jwt handler function, only if it enabled.
func (s *Service) JWT() echo.MiddlewareFunc {
	return s.jwtHandler
}

// JwtToken returns a JWT token as a string. Custom claims can be provided
// in order to be included. Claims `iat“ and `exp` are reserved.
func (s *Service) JwtToken(customClaims map[string]interface{}) string {
	token := jwt.New(jwt.SigningMethodHS512)
	now := time.Now()

	claims := token.Claims.(jwt.MapClaims)
	for k, v := range customClaims {
		claims[k] = v
	}

	claims[common.JwtClaimsIat] = now.Unix()
	claims[common.JwtClaimsExp] = now.Add(s.Configuration().GetHTTP().JwtTokenExpiration).Unix()

	t, _ := token.SignedString([]byte(s.Configuration().GetHTTP().JwtSigningKey))
	return t
}

func (s *Service) initWebServer() {
	s.server = echo.New()

	if s.Configuration().GetApp().LogLevel == int(zap.DebugLevel) {
		s.server.Logger.SetLevel(1)
	} else {
		s.server.Logger.SetLevel(2)
	}

	if !s.Configuration().GetApp().IsDevelopment {
		s.server.HideBanner = true
	}

	// timeouts
	s.server.Server.ReadTimeout = s.Configuration().GetHTTP().ReadTimeout
	s.server.Server.WriteTimeout = s.Configuration().GetHTTP().WriteTimeout
	s.server.Server.IdleTimeout = s.Configuration().GetHTTP().IdleTimeout

	// middlewares
	s.server.Pre(echoMiddleware.RemoveTrailingSlash())
	s.server.Use(echoMiddleware.Recover())
	s.server.Use(middleware.Trace())
	s.server.Use(echoMiddleware.Logger())

	// validator
	s.server.Validator = newValidator()

	// jwt
	if s.Configuration().GetHTTP().JwtEnabled {
		s.jwtHandler = middleware.Jwt([]byte(s.Configuration().GetHTTP().JwtSigningKey))
	}

	// healthcheck
	if !s.Configuration().GetHTTP().CustomHealthCheck {
		s.setHealthCheck()
	}
}

func (s *Service) setHealthCheck() {
	// very basic health check;
	// we have some ideas to improve this with some custom checkers
	s.server.GET("/health", func(c echo.Context) error {
		type healthResponse struct {
			Status string
		}
		resp := healthResponse{Status: "OK"}
		return c.JSON(http.StatusOK, resp)
	})
}
