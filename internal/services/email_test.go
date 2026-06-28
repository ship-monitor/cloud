package services_test

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
)

func TestEmailServiceConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		data              services.EmailServiceConfig
		shouldReturnError bool
	}{
		{data: services.EmailServiceConfig{}, shouldReturnError: true},
		{
			data: services.EmailServiceConfig{
				SMTPHost:     "localhost",
				SMTPPort:     8080,
				AuthPassword: "pass",
				AuthEmail:    "some@mail.com",
				SenderName:   "some name",
			},
		},
	}

	for _, testCase := range testCases {
		err := validator.New().Struct(testCase.data)
		if err == nil && testCase.shouldReturnError {
			t.Fatalf("Validation should fails but there is no error")
		} else if err != nil && !testCase.shouldReturnError {
			t.Fatalf(
				"Validation should be successful, but there error: %s",
				err,
			)
		}
	}
}
