package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"
	"sort"
	"strings"
	"sync"
	"time"
)

type Attachment struct {
	Filename string
	MIMEType string
	Data     []byte
}

type Message struct {
	From        string
	To          []string
	Cc          []string
	Bcc         []string
	Subject     string
	Body        string
	HTML        bool
	Attachments []Attachment
	Date        time.Time
}

func (m *Message) Recipients() []string {
	recipients := make([]string, 0, len(m.To)+len(m.Cc)+len(m.Bcc))
	recipients = append(recipients, m.To...)
	recipients = append(recipients, m.Cc...)
	recipients = append(recipients, m.Bcc...)
	return recipients
}

func Build(msg *Message) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is required")
	}
	if strings.TrimSpace(msg.From) == "" {
		return nil, fmt.Errorf("from is required")
	}
	if len(msg.Recipients()) == 0 {
		return nil, fmt.Errorf("at least one recipient is required")
	}

	headers := textproto.MIMEHeader{}
	headers.Set("From", msg.From)
	if len(msg.To) > 0 {
		headers.Set("To", strings.Join(msg.To, ", "))
	}
	if len(msg.Cc) > 0 {
		headers.Set("Cc", strings.Join(msg.Cc, ", "))
	}
	if msg.Subject != "" {
		headers.Set("Subject", mime.QEncoding.Encode("utf-8", msg.Subject))
	}
	date := msg.Date
	if date.IsZero() {
		date = time.Now()
	}
	headers.Set("Date", date.Format(time.RFC1123Z))
	headers.Set("MIME-Version", "1.0")

	var body bytes.Buffer
	contentType := "text/plain; charset=utf-8"
	if msg.HTML {
		contentType = "text/html; charset=utf-8"
	}

	if len(msg.Attachments) == 0 {
		headers.Set("Content-Type", contentType)
		headers.Set("Content-Transfer-Encoding", "quoted-printable")
	} else {
		writer := multipart.NewWriter(&body)
		partHeader := textproto.MIMEHeader{}
		partHeader.Set("Content-Type", contentType)
		partHeader.Set("Content-Transfer-Encoding", "quoted-printable")
		part, err := writer.CreatePart(partHeader)
		if err != nil {
			return nil, err
		}
		quoted := quotedprintable.NewWriter(part)
		if _, err := quoted.Write([]byte(msg.Body)); err != nil {
			return nil, err
		}
		if err := quoted.Close(); err != nil {
			return nil, err
		}
		for _, attachment := range msg.Attachments {
			if err := writeAttachment(writer, attachment); err != nil {
				return nil, err
			}
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
		headers.Set("Content-Type", "multipart/mixed; boundary="+writer.Boundary())
	}

	var out bytes.Buffer
	writeHeaders(&out, headers)
	out.WriteString("\r\n")
	if len(msg.Attachments) == 0 {
		quoted := quotedprintable.NewWriter(&out)
		if _, err := quoted.Write([]byte(msg.Body)); err != nil {
			return nil, err
		}
		if err := quoted.Close(); err != nil {
			return nil, err
		}
	} else {
		out.Write(body.Bytes())
	}
	return out.Bytes(), nil
}

func writeAttachment(writer *multipart.Writer, attachment Attachment) error {
	mimetype := attachment.MIMEType
	if mimetype == "" {
		mimetype = "application/octet-stream"
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", fmt.Sprintf("%s; name=%q", mimetype, attachment.Filename))
	header.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", attachment.Filename))
	header.Set("Content-Transfer-Encoding", "base64")
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(attachment.Data)
	for len(encoded) > 76 {
		if _, err := part.Write([]byte(encoded[:76] + "\r\n")); err != nil {
			return err
		}
		encoded = encoded[76:]
	}
	if len(encoded) > 0 {
		if _, err := part.Write([]byte(encoded + "\r\n")); err != nil {
			return err
		}
	}
	return nil
}

func writeHeaders(buf *bytes.Buffer, headers textproto.MIMEHeader) {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, value := range headers[key] {
			fmt.Fprintf(buf, "%s: %s\r\n", key, value)
		}
	}
}

type Sender interface {
	Send(ctx context.Context, msg *Message) error
}

type SMTPSender struct {
	Addr     string
	Username string
	Password string
	Host     string
}

func (s *SMTPSender) Send(ctx context.Context, msg *Message) error {
	data, err := Build(msg)
	if err != nil {
		return err
	}
	var auth smtp.Auth
	if s.Username != "" {
		host := s.Host
		if host == "" {
			host = hostFromAddr(s.Addr)
		}
		auth = smtp.PlainAuth("", s.Username, s.Password, host)
	}
	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(s.Addr, auth, msg.From, msg.Recipients(), data)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func hostFromAddr(addr string) string {
	if idx := strings.LastIndex(addr, ":"); idx > 0 {
		return addr[:idx]
	}
	return addr
}

type MemorySender struct {
	mu       sync.Mutex
	Messages []*Message
}

func (m *MemorySender) Send(ctx context.Context, msg *Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, msg)
	return nil
}
