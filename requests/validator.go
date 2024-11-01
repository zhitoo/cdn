package requests

import (
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

type Validator struct {
	validator *validator.Validate
}

type ErrorResponse struct {
	Error       bool
	FailedField string
	Tag         string
	Value       interface{}
}

func NewValidator() *Validator {
	NewValidator := &Validator{
		validator: validate,
	}
	NewValidator.validator.RegisterValidation("teener", func(fl validator.FieldLevel) bool {
		return fl.Field().Uint() >= 12 && fl.Field().Uint() <= 18
	})

	NewValidator.validator.RegisterValidation("mobile", func(fl validator.FieldLevel) bool {
		return len(fl.Field().String()) == 11
	})
	//you can add more custom validator here!!
	return NewValidator
}

func (v Validator) Validate(data interface{}) []ErrorResponse {
	validationErrors := []ErrorResponse{}
	errs := validate.Struct(data)
	if errs != nil {
		for _, err := range errs.(validator.ValidationErrors) {
			var elem ErrorResponse
			elem.FailedField = err.Field() // Export struct field name
			elem.Tag = err.Tag()           // Export struct tag
			elem.Value = err.Value()       // Export field value
			elem.Error = true
			validationErrors = append(validationErrors, elem)
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors
	}
	return nil
}
