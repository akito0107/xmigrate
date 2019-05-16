package xmigrate

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/akito0107/xsqlparser"
	"github.com/akito0107/xsqlparser/dialect"
	"github.com/akito0107/xsqlparser/sqlast"
	"github.com/jmoiron/sqlx"
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

type TableDef struct {
	Name    string
	Columns map[string]*sqlast.SQLColumnDef
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

func (p *PGDump) Dump(ctx context.Context) ([]*TableDef, error) {
	tableNames, err := p.getTableNames(ctx)
	if err != nil {
		return nil, errors.Errorf("getTableNames failed: %w", err)
	}

	var tables []*TableDef
	keymap := make(map[string]*columnInfo)

	for _, n := range tableNames {
		constrains, err := p.getTableConstrains(ctx, n, keymap)
		if err != nil {
			return nil, errors.Errorf("getTableConstrains failed: %w", err)
		}

		columns, err := p.getColumnDefinition(ctx, n)
		if err != nil {
			return nil, errors.Errorf("getColumnDefinition failed: %w", err)
		}

		for _, c := range columns {
			cnts, ok := constrains[c.Name.ToSQLString()]
			if ok {
				c.Constraints = append(c.Constraints, cnts...)
			}
		}

		tables = append(tables, &TableDef{
			Name:    n,
			Columns: columns,
		})
	}

	for _, t := range tables {
		for _, c := range t.Columns {
			for _, ref := range c.Constraints {
				spec, ok := ref.Spec.(*sqlast.ReferencesColumnSpec)
				if !ok {
					continue
				}
				targinfo, ok := keymap[ref.Name.ToSQLString()]
				if !ok {
					return nil, errors.Errorf("key %s is not found", ref.Name.ToSQLString())
				}

				spec.TableName = sqlast.NewSQLObjectName(targinfo.TableName)
				spec.Columns = []*sqlast.SQLIdent{sqlast.NewSQLIdent(targinfo.ColumnName)}
			}
		}
	}

	return tables, nil
}

type pgInformationSchemaTables struct {
	TableCatalog              string         `db:"table_catalog"`
	TableSchema               string         `db:"table_schema"`
	TableName                 string         `db:"table_name"`
	TableType                 string         `db:"table_type"`
	SelfReferencingColumnName sql.NullString `db:"self_referencing_column_name"`
	ReferenceGeneration       sql.NullString `db:"reference_generation"`
	UserDefinedTypeCatalog    sql.NullString `db:"user_defined_type_catalog"`
	UserDefinedTypeSchema     sql.NullString `db:"user_defined_type_schema"`
	UserDefinedTypeName       sql.NullString `db:"user_defined_type_name"`
	IsInsertableInto          string         `db:"is_insertable_into"`
	IsTyped                   string         `db:"is_typed"`
	CommitAction              sql.NullString `db:"commit_action"`
}

func (p *PGDump) getTableNames(ctx context.Context) ([]string, error) {
	var tables []pgInformationSchemaTables
	if err := p.db.SelectContext(ctx, &tables, "select * from information_schema.tables t where t.table_schema = 'public' order by table_name"); err != nil {
		return nil, errors.Errorf("selectContext failed: %w", err)
	}

	var tableNames []string

	for _, t := range tables {
		tableNames = append(tableNames, t.TableName)
	}
	return tableNames, nil
}

type pgInformationSchemaColumns struct {
	TableCatalog           string         `db:"table_catalog"`
	TableSchema            string         `db:"table_schema"`
	TableName              string         `db:"table_name"`
	ColumnName             string         `db:"column_name"`
	OrdinalPosition        int            `db:"ordinal_position"`
	ColumnDefault          sql.NullString `db:"column_default"`
	IsNullable             string         `db:"is_nullable"`
	DataType               string         `db:"data_type"`
	CharacterMaximumLength sql.NullInt64  `db:"character_maximum_length"`
	CharacterOctetLength   sql.NullInt64  `db:"character_octet_length"`
	NumericPrecision       sql.NullInt64  `db:"numeric_precision"`
	NumericPrecisionRadix  sql.NullInt64  `db:"numeric_precision_radix"`
	NumericScale           sql.NullInt64  `db:"numeric_scale"`
	DatetimePrecision      sql.NullInt64  `db:"datetime_precision"`
	IntervalType           sql.NullString `db:"interval_type"`
	IntervalPrecision      sql.NullInt64  `db:"interval_precision"`
	CharacterSetCatalog    sql.NullString `db:"character_set_catalog"`
	CharacterSetSchema     sql.NullString `db:"character_set_schema"`
	CharacterSetName       sql.NullString `db:"character_set_name"`
	CollationCatalog       sql.NullString `db:"collation_catalog"`
	CollationSchema        sql.NullString `db:"collation_schema"`
	CollationName          sql.NullString `db:"collation_name"`
	DomainCatalog          sql.NullString `db:"domain_catalog"`
	DomainSchema           sql.NullString `db:"domain_schema"`
	DomainName             sql.NullString `db:"domain_name"`
	UDTCatalog             sql.NullString `db:"udt_catalog"`
	UDTSchema              sql.NullString `db:"udt_schema"`
	UDTName                sql.NullString `db:"udt_name"`
	ScopeCatalog           sql.NullString `db:"scope_catalog"`
	ScopeSchema            sql.NullString `db:"scope_schema"`
	ScopeName              sql.NullString `db:"scope_name"`
	MaximumCardinality     sql.NullInt64  `db:"maximum_cardinality"`
	DTDIdentifier          sql.NullString `db:"dtd_identifier"`
	IsSelfReferencing      sql.NullString `db:"is_self_referencing"`
	IsIdentity             sql.NullString `db:"is_identity"`
	IdentityGeneration     sql.NullString `db:"identity_generation"`
	IdentityStart          sql.NullString `db:"identity_start"`
	IdentityIncrement      sql.NullString `db:"identity_increment"`
	IdentityMaximum        sql.NullString `db:"identity_maximum"`
	IdentityMinimum        sql.NullString `db:"identity_minimum"`
	IdentityCycle          sql.NullString `db:"identity_cycle"`
	IsGenerated            sql.NullString `db:"is_generated"`
	GenerationExpression   sql.NullString `db:"generation_expression"`
	IsUpdatable            string         `db:"is_updatable"`
}

func (p *PGDump) getColumnDefinition(ctx context.Context, schemaName string) (map[string]*sqlast.SQLColumnDef, error) {
	var columns []*pgInformationSchemaColumns
	if err := p.db.SelectContext(ctx, &columns, "select * from information_schema.columns where table_schema = 'public' and table_name = $1", schemaName); err != nil {
		return nil, errors.Errorf("select columns with tableName %s failed: %w", schemaName, err)
	}

	columndefs := make(map[string]*sqlast.SQLColumnDef)
	for _, c := range columns {
		p := getParser(c.DataType)
		tp, err := p.ParseDataType()
		if err != nil {
			return nil, errors.Errorf("parseDataTypeFailed: %w", err)
		}
		tp = parseTypeOption(tp, c)

		var def sqlast.ASTNode
		if c.ColumnDefault.Valid {
			p = getParser(c.ColumnDefault.String)
			d, err := p.ParseExpr()
			if err != nil {
				return nil, errors.Errorf("parseDefault Value failed: %w", err)
			}
			def = d
		}

		var constrains []*sqlast.ColumnConstraint

		if strings.EqualFold(c.IsNullable, "NO") {
			constrains = append(constrains, &sqlast.ColumnConstraint{
				Spec: &sqlast.NotNullColumnSpec{},
			})
		}

		columndefs[c.ColumnName] = &sqlast.SQLColumnDef{
			DataType:    tp,
			Default:     def,
			Name:        sqlast.NewSQLIdent(c.ColumnName),
			Constraints: constrains,
		}
	}

	return columndefs, nil
}

func parseTypeOption(tp sqlast.SQLType, info *pgInformationSchemaColumns) sqlast.SQLType {
	switch tp.(type) {
	case *sqlast.VarcharType:
		if info.CharacterMaximumLength.Valid {
			return &sqlast.VarcharType{
				Size: sqlast.NewSize(uint8(info.CharacterMaximumLength.Int64)),
			}
		}
		return tp
	default:
		return tp
	}
}

type pgInformationSchemaConstraints struct {
	ColumnName           string         `db:"column_name"`
	ConstraintName       string         `db:"constraint_name"`
	ConstraintType       string         `db:"constraint_type"`
	TableName            string         `db:"table_name"`
	UniqueConstraintName sql.NullString `db:"unique_constraint_name"`
}

type columnInfo struct {
	TableName  string
	ColumnName string
}

func (p *PGDump) getTableConstrains(ctx context.Context, tableName string, keymap map[string]*columnInfo) (map[string][]*sqlast.ColumnConstraint, error) {

	var constrains []*pgInformationSchemaConstraints

	err := p.db.SelectContext(ctx, &constrains,
		`select 
					constraint_column_usage.column_name, 
					table_constraints.constraint_name,
					table_constraints.constraint_type,
					table_constraints.table_name,
					referential_constraints.unique_constraint_name
 				from
					information_schema.table_constraints
				left join information_schema.referential_constraints on referential_constraints.constraint_name = table_constraints.constraint_name
 				join information_schema.constraint_column_usage on constraint_column_usage.constraint_name = table_constraints.constraint_name
 				where table_constraints.table_schema = 'public' and table_constraints.table_name = $1`, tableName)

	if err != nil {
		return nil, errors.Errorf("selectContext failed: %w", err)
	}

	constraintmap := make(map[string][]*sqlast.ColumnConstraint)

	for _, c := range constrains {
		keymap[c.ConstraintName] = &columnInfo{
			TableName:  c.TableName,
			ColumnName: c.ColumnName,
		}

		switch c.ConstraintType {
		case "FOREIGN KEY":
			constraintmap[c.ColumnName] = append(constraintmap[c.ColumnName], &sqlast.ColumnConstraint{
				Name: sqlast.NewSQLIdentifier(sqlast.NewSQLIdent(c.ConstraintName)),
				Spec: &sqlast.ReferencesColumnSpec{},
			})
		case "UNIQUE":
			constraintmap[c.ColumnName] = append(constraintmap[c.ColumnName], &sqlast.ColumnConstraint{
				Name: sqlast.NewSQLIdentifier(sqlast.NewSQLIdent(c.ConstraintName)),
				Spec: &sqlast.UniqueColumnSpec{},
			})
		case "PRIMARY KEY":
			constraintmap[c.ColumnName] = append(constraintmap[c.ColumnName], &sqlast.ColumnConstraint{
				Name: sqlast.NewSQLIdentifier(sqlast.NewSQLIdent(c.ConstraintName)),
				Spec: &sqlast.UniqueColumnSpec{
					IsPrimaryKey: true,
				},
			})
		default:
			return nil, errors.Errorf("currently unsupported constraint %s", c.ConstraintType)
		}
	}

	return constraintmap, nil
}

func getParser(src string) *xsqlparser.Parser {
	parser, err := xsqlparser.NewParser(bytes.NewBufferString(src), &dialect.PostgresqlDialect{})
	if err != nil {
		log.Fatalf("initialize parser failed with input %s err: %+v", src, err)
	}

	return parser
}
