package email

type Email struct {
	To      string `validate:"required,gt=0"`
	Subject string `validate:"required"`
	Body    []byte `validate:"required"`
	Lang    string `validate:"required"`
}

type Sender struct {
	Email string `validate:"required,email"`
	Name  string `validate:"required"`
}
