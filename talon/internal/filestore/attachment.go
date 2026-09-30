package filestore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ajepe/talon/internal/orm"
	"github.com/ajepe/talon/internal/rpc"
)

type AttachmentService struct {
	store *Store
	db    *sql.DB
}

func NewAttachmentService(store *Store, db *sql.DB) *AttachmentService {
	if store == nil {
		return nil
	}
	return &AttachmentService{store: store, db: db}
}

func (s *AttachmentService) Get(ctx context.Context, attachmentID int) ([]byte, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("no filestore configured")
	}
	if s.db == nil {
		return nil, fmt.Errorf("no database connection")
	}
	var fileID sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT file_id FROM ir_attachment WHERE id = $1`, attachmentID).Scan(&fileID)
	if err != nil {
		return nil, err
	}
	if !fileID.Valid || fileID.String == "" {
		return nil, nil
	}
	return s.store.Get(fileID.String)
}

func (s *AttachmentService) Set(ctx context.Context, attachmentID int, data []byte) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("no filestore configured")
	}
	if s.db == nil {
		return fmt.Errorf("no database connection")
	}
	name, err := s.store.Save(data)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE ir_attachment SET file_id = $1, size = $2 WHERE id = $3`,
		name, len(data), attachmentID)
	return err
}

func (s *AttachmentService) Delete(ctx context.Context, attachmentID int) error {
	if s == nil || s.store == nil {
		return nil
	}
	if s.db == nil {
		return fmt.Errorf("no database connection")
	}
	var fileID sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT file_id FROM ir_attachment WHERE id = $1`, attachmentID).Scan(&fileID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if fileID.Valid && fileID.String != "" {
		if err := s.store.Delete(fileID.String); err != nil {
			return err
		}
	}
	return nil
}

func (s *AttachmentService) RegisterRPC(srv *rpc.Server) {
	if s == nil {
		return
	}
	srv.Register("model.ir.attachment.get_data", func(ctx context.Context, params []interface{}) (interface{}, error) {
		if len(params) < 1 {
			return nil, &rpc.UserError{Message: "get_data requires an attachment id"}
		}
		id, ok := orm.AsInt(params[0])
		if !ok {
			return nil, &rpc.UserError{Message: "invalid attachment id"}
		}
		data, err := s.Get(ctx, id)
		if err != nil {
			return nil, &rpc.UserError{Message: err.Error()}
		}
		return data, nil
	})

	srv.Register("model.ir.attachment.set_data", func(ctx context.Context, params []interface{}) (interface{}, error) {
		if len(params) < 2 {
			return nil, &rpc.UserError{Message: "set_data requires an attachment id and data"}
		}
		id, ok := orm.AsInt(params[0])
		if !ok {
			return nil, &rpc.UserError{Message: "invalid attachment id"}
		}
		data, _ := params[1].([]byte)
		if data == nil {
			if text, ok := params[1].(string); ok {
				data = []byte(text)
			}
		}
		if err := s.Set(ctx, id, data); err != nil {
			return nil, &rpc.UserError{Message: err.Error()}
		}
		return nil, nil
	})
}
