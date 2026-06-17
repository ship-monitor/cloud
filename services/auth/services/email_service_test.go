package services

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestEmailServiceConfig(t *testing.T) {
	testCases := []struct {
		data              EmailServiceConfig
		shouldReturnError bool
	}{
		{data: EmailServiceConfig{}, shouldReturnError: true},
		{
			data: EmailServiceConfig{
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
			t.Fatalf("Validation should be successful, but there error: %s", err)
		}
	}
}
