package morondanga

import "gopkg.in/go-playground/validator.v9"

type Validator struct {
	validator *validator.Validate
}

func (v Validator) Validate(i interface{}) error {
	return v.validator.Struct(i)
}

func newValidator() *Validator {
	return &Validator{
		validator: validator.New(),
	}
}
