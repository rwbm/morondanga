package common

import (
	"errors"

	"github.com/labstack/echo/v4"
)

const (
	ERROR_KEY_MESSAGE = "message"
)

var (
	ErrJWTMissing              = errors.New("missing_jwt")
	ErrJWTInvalid              = errors.New("invalid_jwt")
	ErrJWTInvalidSigningMethod = errors.New("invalid_jwt_signing")
	ErrJWTInvalidAlgorithm     = errors.New("invalid_jwt_algo")

	ErrInvalidRequest = errors.New("invalid_request")
)

type Error struct {
	Error map[string]interface{} `json:"error"`
}

func NewError(err error) Error {
	e := Error{}
	e.Error = make(map[string]interface{})

	switch v := err.(type) {
	case *echo.HTTPError:
		e.Error[ERROR_KEY_MESSAGE] = v.Message
	default:
		e.Error[ERROR_KEY_MESSAGE] = v.Error()
	}
	return e
}

func NewErrorMessage(s string) Error {
	e := Error{}
	e.Error = make(map[string]interface{})
	e.Error[ERROR_KEY_MESSAGE] = s
	return e
}
