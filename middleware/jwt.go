package middleware

import (
	"net/http"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/rwbm/morondanga/common"
)

type (
	jwtConfig struct {
		Skipper    skipper
		SigningKey interface{}
	}
	skipper      func(c echo.Context) bool
	jwtExtractor func(echo.Context) (string, error)
)

func Jwt(key interface{}) echo.MiddlewareFunc {
	c := jwtConfig{}
	c.SigningKey = key
	return JwtWithConfig(c)
}

// JwtWithConfig defines the middleware to extract access token from the request.
func JwtWithConfig(config jwtConfig) echo.MiddlewareFunc {
	extractor := jwtFromHeader("Authorization", "Bearer")
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth, err := extractor(c)
			if err != nil {
				if config.Skipper != nil {
					if config.Skipper(c) {
						return next(c)
					}
				}
				return c.JSON(http.StatusUnauthorized, common.NewError(err))
			}

			token, err := jwt.Parse(auth, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, common.ErrJWTInvalidSigningMethod
				}
				return config.SigningKey, nil
			})
			if err != nil {
				return c.JSON(http.StatusForbidden, common.NewError(common.ErrJWTMissing))
			}

			// validate algorithm
			if token.Method.Alg() != "HS512" {
				return c.JSON(http.StatusForbidden, common.NewError(common.ErrJWTInvalidAlgorithm))
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				// put all the claims in the context
				for k, v := range claims {
					if k != "iat" && k != "exp" {
						c.Set(k, v)
					}
				}
				return next(c)
			}
			return c.JSON(http.StatusForbidden, common.NewError(common.ErrJWTInvalid))
		}
	}
}

// jwtFromHeader returns a `jwtExtractor` that extracts token from the request header.
func jwtFromHeader(header string, authScheme string) jwtExtractor {
	return func(c echo.Context) (string, error) {
		auth := c.Request().Header.Get(header)
		l := len(authScheme)
		if len(auth) > l+1 && auth[:l] == authScheme {
			return auth[l+1:], nil
		}
		return "", common.ErrJWTInvalid
	}
}
