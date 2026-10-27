package middleware

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/rwbm/morondanga/common"
)

type (
	jwtConfig struct {
		Skipper    skipper
		SigningKey interface{}
		// CookieName is the cookie the token is read from when the request has no
		// Authorization header; none if empty.
		CookieName string
	}
	skipper      func(c echo.Context) bool
	jwtExtractor func(echo.Context) (string, error)
)

func Jwt(key interface{}) echo.MiddlewareFunc {
	c := jwtConfig{}
	c.SigningKey = key
	return JwtWithConfig(c)
}

// JwtWithCookie reads the token from the Authorization header or, without it,
// from the cookie.
func JwtWithCookie(key interface{}, cookieName string) echo.MiddlewareFunc {
	return JwtWithConfig(jwtConfig{SigningKey: key, CookieName: cookieName})
}

// JwtWithConfig defines the middleware to extract access token from the request.
func JwtWithConfig(config jwtConfig) echo.MiddlewareFunc {
	extractor := jwtFromHeader(common.HttpHeaderAuthprization, "Bearer")
	if config.CookieName != "" {
		extractor = jwtFromHeaderOrCookie(extractor, config.CookieName)
	}
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
			// an expired or forged token is a missing authentication: 401, for the
			// client to authenticate again
			if err != nil {
				return c.JSON(http.StatusUnauthorized, common.NewError(common.ErrJWTMissing))
			}

			// validate algorithm
			if token.Method.Alg() != common.JwtAlgoHS512 {
				return c.JSON(http.StatusUnauthorized, common.NewError(common.ErrJWTInvalidAlgorithm))
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				// put all the claims in the context
				for k, v := range claims {
					if k != common.JwtClaimsIat && k != common.JwtClaimsExp {
						c.Set(k, v)
					}
				}
				return next(c)
			}
			return c.JSON(http.StatusUnauthorized, common.NewError(common.ErrJWTInvalid))
		}
	}
}

// jwtFromHeaderOrCookie returns a `jwtExtractor` that falls back to the cookie
// when the header has no token.
func jwtFromHeaderOrCookie(fromHeader jwtExtractor, cookieName string) jwtExtractor {
	return func(c echo.Context) (string, error) {
		if auth, err := fromHeader(c); err == nil {
			return auth, nil
		}
		cookie, err := c.Cookie(cookieName)
		if err != nil || cookie.Value == "" {
			return "", common.ErrJWTInvalid
		}
		return cookie.Value, nil
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
