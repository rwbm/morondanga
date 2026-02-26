package middleware

import (
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

type RateLimitConfig = echoMiddleware.RateLimiterConfig
type RateLimitStore = echoMiddleware.RateLimiterStore
type RateLimitMemoryStoreConfig = echoMiddleware.RateLimiterMemoryStoreConfig

func RateLimit(store RateLimitStore) echo.MiddlewareFunc {
	return echoMiddleware.RateLimiter(store)
}

func RateLimitWithConfig(config RateLimitConfig) echo.MiddlewareFunc {
	return echoMiddleware.RateLimiterWithConfig(config)
}

func NewRateLimitMemoryStore(limit rate.Limit) *echoMiddleware.RateLimiterMemoryStore {
	return echoMiddleware.NewRateLimiterMemoryStore(limit)
}

func NewRateLimitMemoryStoreWithConfig(config RateLimitMemoryStoreConfig) *echoMiddleware.RateLimiterMemoryStore {
	return echoMiddleware.NewRateLimiterMemoryStoreWithConfig(config)
}
