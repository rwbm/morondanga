package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

var testKey = []byte("a-signing-key-of-at-least-32-bytes")

func signed(t *testing.T, exp time.Time) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{"sub": "usr-1", "exp": exp.Unix()})
	s, err := token.SignedString(testKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func run(t *testing.T, cookieName string, prepare func(*http.Request)) (int, any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	prepare(req)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	var sub any
	err := JwtWithConfig(jwtConfig{SigningKey: testKey, CookieName: cookieName})(func(c echo.Context) error {
		sub = c.Get("sub")
		return c.NoContent(http.StatusNoContent)
	})(c)
	if err != nil {
		t.Fatalf("middleware: %v", err)
	}
	return rec.Code, sub
}

func TestTheTokenIsReadFromTheAuthorizationHeader(t *testing.T) {
	code, sub := run(t, "access_token", func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+signed(t, time.Now().Add(time.Hour)))
	})

	if code != http.StatusNoContent || sub != "usr-1" {
		t.Fatalf("got %d with sub %v", code, sub)
	}
}

func TestTheTokenIsReadFromTheCookieWithoutAuthorization(t *testing.T) {
	code, sub := run(t, "access_token", func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: "access_token", Value: signed(t, time.Now().Add(time.Hour))})
	})

	if code != http.StatusNoContent || sub != "usr-1" {
		t.Fatalf("got %d with sub %v", code, sub)
	}
}

func TestTheCookieIsIgnoredWithoutACookieName(t *testing.T) {
	code, _ := run(t, "", func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: "access_token", Value: signed(t, time.Now().Add(time.Hour))})
	})

	if code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", code)
	}
}

// An expired or forged token is a missing authentication (401), for the client
// to refresh it, not a missing permission (403).
func TestAnExpiredOrForgedTokenIsUnauthorized(t *testing.T) {
	forged := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{"sub": "usr-1", "exp": time.Now().Add(time.Hour).Unix()})
	forgedToken, _ := forged.SignedString([]byte("another-key-of-at-least-32-bytes!!"))

	for name, token := range map[string]string{
		"expired": signed(t, time.Now().Add(-time.Minute)),
		"forged":  forgedToken,
		"garbage": "not-a-jwt",
	} {
		code, _ := run(t, "access_token", func(r *http.Request) {
			r.Header.Set("Authorization", "Bearer "+token)
		})
		if code != http.StatusUnauthorized {
			t.Fatalf("%s: expected 401, got %d", name, code)
		}
	}
}
