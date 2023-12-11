package common

import (
	"errors"

	"github.com/labstack/echo/v4"
)

var (
	ErrJWTMissing              = errors.New("missing_jwt")
	ErrJWTInvalid              = errors.New("invalid_jwt")
	ErrJWTInvalidSigningMethod = errors.New("invalid_jwt_signing")
	ErrJWTInvalidAlgorithm     = errors.New("invalid_jwt_algo")
)

type Error struct {
	Error map[string]interface{} `json:"error"`
}

func NewError(err error) Error {
	e := Error{}
	e.Error = make(map[string]interface{})

	switch v := err.(type) {
	case *echo.HTTPError:
		e.Error["message"] = v.Message
	default:
		e.Error["message"] = v.Error()
	}
	return e
}
