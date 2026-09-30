package email

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"testing"
	"time"
)

func TestBuildSimple(t *testing.T) {
	raw, err := Build(&Message{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Hello",
		Body:    "Hello world",
		Date:    time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if msg.Header.Get("From") != "sender@example.com" {
		t.Errorf("From = %q", msg.Header.Get("From"))
	}
	if msg.Header.Get("Subject") != "Hello" {
		t.Errorf("Subject = %q", msg.Header.Get("Subject"))
	}
	body, _ := io.ReadAll(msg.Body)
	if strings.TrimSpace(string(body)) != "Hello world" {
		t.Errorf("body = %q, want Hello world", body)
	}
}

func TestBuildEncodesNonASCIISubject(t *testing.T) {
	raw, err := Build(&Message{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Réunion",
		Body:    "body",
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	msg, _ := mail.ReadMessage(strings.NewReader(string(raw)))
	decoded, err := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if err != nil {
		t.Fatalf("DecodeHeader() error = %v", err)
	}
	if decoded != "Réunion" {
		t.Errorf("subject = %q, want Réunion", decoded)
	}
}

func TestBuildWithAttachment(t *testing.T) {
	raw, err := Build(&Message{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Report",
		Body:    "See attachment",
		Attachments: []Attachment{
			{Filename: "report.csv", MIMEType: "text/csv", Data: []byte("Name,Code\nAcme,ACME\n")},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("ParseMediaType() error = %v", err)
	}
	if !strings.HasPrefix(mediaType, "multipart/mixed") {
		t.Fatalf("content type = %q, want multipart/mixed", mediaType)
	}

	reader := multipart.NewReader(msg.Body, params["boundary"])
	var attachmentData string
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("NextPart() error = %v", err)
		}
		disposition, _, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
		if disposition != "attachment" {
			continue
		}
		data, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		attachmentData = string(data)
	}
	if !strings.Contains(attachmentData, "TmFtZSxDb2Rl") {
		t.Errorf("attachment data = %q, missing base64 content", attachmentData)
	}
}

func TestBuildValidation(t *testing.T) {
	if _, err := Build(&Message{To: []string{"a@b.c"}}); err == nil {
		t.Error("Build() expected error for missing from")
	}
	if _, err := Build(&Message{From: "a@b.c"}); err == nil {
		t.Error("Build() expected error for missing recipients")
	}
}

func TestMemorySender(t *testing.T) {
	sender := &MemorySender{}
	msg := &Message{From: "a@b.c", To: []string{"c@d.e"}, Body: "hi"}
	if err := sender.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if len(sender.Messages) != 1 {
		t.Errorf("Messages = %d, want 1", len(sender.Messages))
	}
}

func TestHostFromAddr(t *testing.T) {
	if got := hostFromAddr("smtp.example.com:587"); got != "smtp.example.com" {
		t.Errorf("hostFromAddr() = %q, want smtp.example.com", got)
	}
}
