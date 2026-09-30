package bus

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type PGNotifier struct {
	conn *pgx.Conn
	hub  *Hub
}

func ConnectPG(ctx context.Context, uri string, hub *Hub) (*PGNotifier, error) {
	conn, err := pgx.Connect(ctx, uri)
	if err != nil {
		return nil, err
	}
	return &PGNotifier{conn: conn, hub: hub}, nil
}

func (p *PGNotifier) Listen(ctx context.Context, channel string) error {
	_, err := p.conn.Exec(ctx, "LISTEN "+quoteChannel(channel))
	return err
}

func (p *PGNotifier) Notify(ctx context.Context, channel string, data map[string]interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := p.conn.Exec(ctx, "SELECT pg_notify($1, $2)", channel, string(payload)); err != nil {
		return err
	}
	p.hub.Publish(channel, Message{Type: "notification", Data: data})
	return nil
}

func (p *PGNotifier) Run(ctx context.Context) error {
	for {
		notification, err := p.conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		p.publish(notification)
	}
}

func (p *PGNotifier) publish(notification *pgconn.Notification) {
	data := map[string]interface{}{}
	if notification.Payload != "" {
		_ = json.Unmarshal([]byte(notification.Payload), &data)
	}
	p.hub.Publish(notification.Channel, Message{Type: "notification", Data: data})
}

func (p *PGNotifier) Close(ctx context.Context) error {
	return p.conn.Close(ctx)
}

func quoteChannel(channel string) string {
	return `"` + strings.ReplaceAll(channel, `"`, `""`) + `"`
}
