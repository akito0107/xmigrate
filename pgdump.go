package xmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/k0kubun/pp"
	_ "github.com/lib/pq"
	errors "golang.org/x/xerrors"
)

type PGConf struct {
	DBName     string `env:"DB_NAME"`
	DBHost     string `env:"DB_HOST"`
	DBPort     string `env:"DB_PORT"`
	DBPassword string `env:"DB_PASSWORD"`
	UserName   string `env:"DB_USER_NAME"`
	SSLMode    bool   `env:"DB_SSL_MODE,allow-empty"`
}

func (p *PGConf) Src() string {
	return fmt.Sprintf("user=%s dbname=%s sslmode=disable host=%s password=%s port=%s", p.UserName, p.DBName, p.DBHost, p.DBPassword, p.DBPort)
}

type PGDump struct {
	db *sqlx.DB
}

func NewPGDump(conf *PGConf) *PGDump {
	src := conf.Src()
	db, err := sqlx.Connect("postgres", src)
	if err != nil {
		log.Fatalf("connect failed with src: %s, err: %+v", src, err)
	}

	return &PGDump{db: db}
}

type pgInformationSchemaTables struct {
	TableCatalog              string         `db:"table_catalog"`
	TableSchema               string         `db:"table_schema"`
	TableName                 string         `db:"table_name"`
	TableType                 string         `db:"table_type"`
	SelfReferencingColumnName string         `db:"self_referencing_column_name"`
	ReferenceGeneration       string         `db:"reference_generation"`
	UserDefinedTypeCatalog    sql.NullString `db:"user_defined_type_catalog"`
	UserDefinedTypeSchema     sql.NullString `db:"user_defined_type_schema"`
	UserDefinedTypeName       sql.NullString `db:"user_defined_type_name"`
	IsInsertableInto          string         `db:"is_insertable_into"`
	IsTyped                   string         `db:"is_typed"`
}

func (p *PGDump) Dump(ctx context.Context) error {
	var tables []pgInformationSchemaTables
	if err := p.db.SelectContext(ctx, &tables, "select * from information_schema.tables t where t.table_schema = 'public'"); err != nil {
		return errors.Errorf("selectContext failed: %w", err)
	}

	for _, t := range tables {
		pp.Println(t)
	}

	return nil
}
