package morondanga

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/rwbm/morondanga/common"
	"github.com/rwbm/morondanga/middleware"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const maxBodyLogSize = 64 * 1024

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

	s.server.HideBanner = true
	s.server.Server.ReadTimeout = s.Configuration().GetHTTP().ReadTimeout
	s.server.Server.WriteTimeout = s.Configuration().GetHTTP().WriteTimeout
	s.server.Server.IdleTimeout = s.Configuration().GetHTTP().IdleTimeout

	// middlewares
	s.server.Pre(echoMiddleware.RemoveTrailingSlash())
	s.server.Use(echoMiddleware.Recover())
	// otelecho must be registered before the request logger so the span is
	// already in the context when we log latency + status.
	if obs := s.Configuration().GetObservability(); obs != nil && obs.Enabled {
		s.server.Use(otelecho.Middleware(s.Configuration().GetApp().Name))
		s.server.Use(s.httpMetricsMiddleware())
	}
	s.server.Use(s.httpRequestLogger())
	if s.Configuration().GetHTTP().AddTraceID {
		s.server.Use(middleware.Trace())
	}

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

func (s *Service) httpRequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()

			inFields := []zap.Field{
				zap.String("method", req.Method),
				zap.String("uri", req.RequestURI),
				zap.String("ip", c.RealIP()),
				zap.Any("headers", flatHeaders(req.Header)),
			}
			if req.Body != nil && req.Body != http.NoBody {
				rawBody, _ := io.ReadAll(req.Body)
				req.Body = io.NopCloser(bytes.NewReader(rawBody))
				if len(rawBody) > 0 {
					logBody := rawBody
					if len(logBody) > maxBodyLogSize {
						logBody = logBody[:maxBodyLogSize]
					}
					inFields = append(inFields, zap.String("body", string(logBody)))
				}
			}
			if sc := trace.SpanFromContext(req.Context()).SpanContext(); sc.IsValid() {
				inFields = append(inFields,
					zap.String("trace_id", sc.TraceID().String()),
					zap.String("span_id", sc.SpanID().String()),
				)
			}
			s.log.Info("Incoming request", inFields...)

			resCapture := newResponseCapture(c.Response().Writer)
			c.Response().Writer = resCapture

			start := time.Now()
			err := next(c)

			res := c.Response()
			outFields := []zap.Field{
				zap.String("method", req.Method),
				zap.String("uri", req.RequestURI),
				zap.Int("status", res.Status),
				zap.Duration("latency", time.Since(start)),
				zap.Any("headers", flatHeaders(res.Header())),
			}
			if body := resCapture.bytes(); len(body) > 0 {
				outFields = append(outFields, zap.String("body", string(body)))
			}
			if sc := trace.SpanFromContext(req.Context()).SpanContext(); sc.IsValid() {
				outFields = append(outFields,
					zap.String("trace_id", sc.TraceID().String()),
					zap.String("span_id", sc.SpanID().String()),
				)
			}
			s.log.Info("Outgoing response", outFields...)

			return err
		}
	}
}

// flatHeaders converts http.Header to a flat map for structured logging.
// Authorization and X-API-Key values are redacted.
func flatHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, vals := range h {
		if strings.EqualFold(k, "Authorization") || strings.EqualFold(k, "X-Api-Key") {
			out[k] = "[REDACTED]"
			continue
		}
		if len(vals) > 0 {
			out[k] = vals[0]
		}
	}
	return out
}

type responseCapture struct {
	http.ResponseWriter
	buf *bytes.Buffer
}

func newResponseCapture(w http.ResponseWriter) *responseCapture {
	return &responseCapture{ResponseWriter: w, buf: &bytes.Buffer{}}
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	if rc.buf.Len() < maxBodyLogSize {
		remaining := maxBodyLogSize - rc.buf.Len()
		if len(b) <= remaining {
			rc.buf.Write(b)
		} else {
			rc.buf.Write(b[:remaining])
		}
	}
	return rc.ResponseWriter.Write(b)
}

func (rc *responseCapture) Flush() {
	if f, ok := rc.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rc *responseCapture) bytes() []byte {
	return rc.buf.Bytes()
}

// httpMetricsMiddleware records http.server.request.duration for every request.
// The histogram is created once when the middleware is initialized and reused
// across all requests, keeping the hot path allocation-free.
func (s *Service) httpMetricsMiddleware() echo.MiddlewareFunc {
	meter := otel.Meter(s.Configuration().GetApp().Name)
	reqDuration, _ := meter.Float64Histogram(
		"http.server.request.duration",
		otelmetric.WithUnit("s"),
		otelmetric.WithDescription("Duration of HTTP server requests."),
		otelmetric.WithExplicitBucketBoundaries(
			0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10,
		),
	)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			reqDuration.Record(c.Request().Context(), time.Since(start).Seconds(),
				otelmetric.WithAttributes(
					attribute.String("http.request.method", c.Request().Method),
					attribute.Int("http.response.status_code", c.Response().Status),
					attribute.String("http.route", c.Path()),
				),
			)
			return err
		}
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
