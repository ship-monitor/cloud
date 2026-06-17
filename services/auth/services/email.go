package services

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type HTMLEmail struct {
	receiverEmail string
	receiverName  string
	title         string
	content       string
}

var _ IEmail = (*HTMLEmail)(nil)

func NewHTMLEmail(receiverAddress, receiverName, title, content string) (*HTMLEmail, error) {
	v := validator.New()
	if err := v.Var(receiverAddress, "email,required"); err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}
	if err := v.Var(receiverName, "required"); err != nil {
		return nil, fmt.Errorf("invalid content: %w", err)
	}
	if err := v.Var(title, "required"); err != nil {
		return nil, fmt.Errorf("invalid title: %w", err)
	}
	if err := v.Var(content, "required"); err != nil {
		return nil, fmt.Errorf("invalid content: %w", err)
	}

	return &HTMLEmail{
		receiverEmail: receiverAddress,
		receiverName:  receiverName,
		content:       content,
		title:         title,
	}, nil
}

func (e HTMLEmail) WriteEmail(senderName, senderEmail string) []byte {
	message := fmt.Sprintf("From: %s <%s>\n", senderName, senderEmail) +
		fmt.Sprintf("To: user <%s>\n", e.receiverEmail) +
		fmt.Sprintf("Subject: %s\n", e.title) +
		fmt.Sprintf("Content-Type: %s\n", ContentTypeHTML) +
		fmt.Sprintf("Message-ID: %s\n", uuid.New().String()) +
		"MIME-Version: 1.0\n" +
		"\n" +
		e.content

	return []byte(message)
}

const ContentTypeHTML = "text/html; charset=UTF-8"

func (e HTMLEmail) ReceiverEmail() string {
	return e.receiverEmail
}
