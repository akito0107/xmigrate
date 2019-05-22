package xmigrate

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	errors "golang.org/x/xerrors"
)

type MigrateLog struct {
	ID        int       `db:"id"`
	MigrateID string    `db:"migrate_id"`
	CreatedAt time.Time `db:"created_at"`
}

func CheckCurrent(ctx context.Context, db *sqlx.DB) (string, error) {
	var t pgInformationSchemaTables
	err := db.GetContext(ctx, &t, "select * from information_schema.tables where table_schema = 'public' and table_name = 'xmigrate';")
	if err == sql.ErrNoRows {
		if _, err := db.ExecContext(ctx, `
create table xmigrate(
  id serial primary key,
  migrate_id varchar not null unique,
  created_at timestamp with time zone default current_timestamp
  )
`); err != nil {
			return "", errors.Errorf("execContext failed: %w", err)
		}
	}
	var logs []MigrateLog

	if err := db.SelectContext(ctx, &logs, "select * from xmigrate order by migrate_id desc limit 1"); err != nil {
		return "", errors.Errorf("selectContext failed: %w", err)
	}

	if len(logs) == 0 {
		return "", nil
	}

	return logs[0].MigrateID, nil
}
