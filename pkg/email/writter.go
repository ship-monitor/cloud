package email

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/textproto"
	"time"

	"github.com/google/uuid"
)

const (
	ContentTypeHTML = "text/html; charset=UTF-8"
	Comment         = "sended via ship system"
)

// Writer writes email as smtp document.
//
// TODO: fix doc.
type Writer interface {
	Write(e Email) ([]byte, error)
}

var _ Writer = (*HTMLWriter)(nil)

type HTMLWriter struct {
	sender Sender
}

func NewHTMLWriter(sender Sender) *HTMLWriter {
	return &HTMLWriter{
		sender: sender,
	}
}

func (w *HTMLWriter) Write(e Email) ([]byte, error) {
	lang := "en"

	encodedSubject := mime.QEncoding.Encode("utf-8", e.Subject)

	headers := textproto.MIMEHeader{}
	headers.Add("Content-Type", ContentTypeHTML)
	headers.Add("Subject", encodedSubject)
	headers.Add("From", fmt.Sprintf("%s <%s>", w.sender.Name, w.sender.Email))
	headers.Add("Comments", Comment)
	headers.Add("Language", lang)         // TODO: fix this
	headers.Add("Content-Language", lang) // TODO: fix this
	headers.Add("Date", time.Now().String())
	headers.Add("Message-ID", uuid.NewString())
	headers.Add("To", e.To)
	headers.Add("MIME-Version", "1.0")

	var buf bytes.Buffer

	err := writeHeaders(&buf, headers)
	if err != nil {
		return nil, fmt.Errorf("write headers: %w", err)
	}

	_, err = fmt.Fprintln(&buf)
	if err != nil {
		return nil, fmt.Errorf("write headers separator: %w", err)
	}

	_, err = fmt.Fprint(&buf, string(e.Body))
	if err != nil {
		return nil, fmt.Errorf("write body: %w", err)
	}

	return buf.Bytes(), nil
}

func writeHeaders(w io.Writer, headers textproto.MIMEHeader) error {
	for h, val := range headers {
		for _, v := range val {
			_, err := fmt.Fprintf(w, "%s: %s\n", h, v)
			if err != nil {
				return fmt.Errorf("write header %q: %w", h, err)
			}
		}
	}

	return nil
}
