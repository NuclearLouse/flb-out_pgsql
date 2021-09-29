package pgclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
)

const (
	defaultSchema string = "public"
)

type PGConfig struct {
	Schema     string
	Table      string
	connString string
}

type PGClient struct {
	Config *PGConfig
	pool   *pgxpool.Pool
	log    *logrus.Logger
}

func NewConfig(connString, schema, table string) *PGConfig {

	if schema == "" {
		schema = defaultSchema
	}
	return &PGConfig{
		Schema:     schema,
		Table:      table,
		connString: connString,
	}
}

func New(ctx context.Context, config *PGConfig, log *logrus.Logger) (*PGClient, error) {
	pool, err := pgxpool.Connect(ctx, config.connString)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create connection pool")
	}

	return &PGClient{
		Config: config,
		pool:   pool,
		log:    log,
	}, nil
}

func (pg *PGClient) CheckIfTableExist(ctx context.Context) error {
	tx, err := pg.pool.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to begin a transaction")
	}
	defer tx.Rollback(ctx)

	var dbName, schemaName, schema string

	//TODO: Всетаки попробовать создавать таблицы и схему через IF NOT EXISTS
	//&& err.Error() != "no rows in result set"

	pg.log.Tracef("check if schema %s exists", pg.Config.Schema)

	if err := tx.QueryRow(ctx,
		"SELECT catalog_name FROM information_schema.schemata WHERE schema_name = $1", pg.Config.Schema,
	).Scan(&dbName); err != nil {
		switch err {
		case pgx.ErrNoRows:
			schema = "NOT EXISTS"
		default:
			return errors.Wrap(err, "unable to scan datas from information_schema.schemata")
		}
	}

	if schema == "NOT EXISTS" {
		if _, err := tx.Exec(ctx,
			fmt.Sprintf("CREATE SCHEMA %s", pg.Config.Schema),
		); err != nil {
			return errors.Wrap(err, "unable to create schema")
		}

		pg.log.Tracef("[pg-plugin] schema %s not exists - create schema", pg.Config.Schema)

	} else {

		pg.log.Tracef("[pg-plugin] skip creating a schema - the schema: %s exists", pg.Config.Schema)

		rows, err := tx.Query(ctx,
			"SELECT table_schema FROM information_schema.tables WHERE table_name = $1", pg.Config.Table,
		)
		if err != nil {
			return errors.Wrap(err, "unable to check information_schema.tables")
		}
		defer rows.Close()
		tmap := make(map[string]string)
		for rows.Next() {
			if err := rows.Scan(&schemaName); err != nil {
				return errors.Wrap(err, "unable to scan datas from information_schema.tables")
			}
			tmap[schemaName] = pg.Config.Table
		}

		if _, ok := tmap[pg.Config.Schema]; ok {
			rows, err := tx.Query(ctx,
				"select column_name, data_type from information_schema.columns where table_schema = $1 and table_name = $2",
				pg.Config.Schema, pg.Config.Table)
			if err != nil {
				return errors.Wrap(err, "unable to check information_schema.columns")
			}
			defer rows.Close()
			cmap := make(map[string]string)
			for rows.Next() {
				var columnName, dataType string
				if err := rows.Scan(&columnName, &dataType); err != nil {
					return errors.Wrap(err, "unable to scan datas from information_schema.columns")
				}
				cmap[columnName] = dataType
			}
			// tag = character varying
			// time = timestamp without time zone
			// data = jsonb
			type1, ok1 := cmap["tag"]
			type2, ok2 := cmap["time"]
			type3, ok3 := cmap["data"]
			if ok1 && ok2 && ok3 {
				if type1 == "character varying" && type2 == "timestamp without time zone" && type3 == "jsonb" {
					if err := tx.Commit(ctx); err != nil {
						return errors.Wrap(err, "unable to commit transaction")
					}

					pg.log.Tracef("[pg-plugin] skip creating a table - the table: %s exists", pg.Config.Table)

					return nil
				}
			}
			return errors.Wrap(err, "database has exist similar table name")
		}

	}

	if _, err := tx.Exec(ctx,
		fmt.Sprintf(`CREATE TABLE %s.%s (tag varchar NULL,"time" timestamp NULL,"data" jsonb NULL)`,
			pg.Config.Schema,
			pg.Config.Table),
	); err != nil {
		return errors.Wrap(err, "unable to create table")
	}

	pg.log.Traceln("[pg-plugin] create table:", pg.Config.Table)

	if err := tx.Commit(ctx); err != nil {
		return errors.Wrap(err, "unable to commit transaction")
	}
	return nil
}

func (pg *PGClient) FlushLogs(ctx context.Context, tag string, datas []json.RawMessage) error {
	db, err := pg.pool.Acquire(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to acquire a database connection from the pool")
	}
	defer db.Release()

	query := fmt.Sprintf(`INSERT INTO %s.%s (tag, "time", "data") VALUES($1, $2, $3)`,
		pg.Config.Schema,
		pg.Config.Table)

	if _, err := db.Conn().Prepare(ctx, "insert logs", query); err != nil {
		return errors.Wrap(err, "unable to prepare batch insert ")
	}

	var data pgtype.JSON
	batch := &pgx.Batch{}

	for _, d := range datas {
		data.Set(d)
		batch.Queue("insert logs", tag, time.Now(), data.Bytes)
	}
	br := db.SendBatch(ctx, batch)
	if err := br.Close(); err != nil {
		return errors.Wrap(err, "unable to close batch insert ")
	}

	return nil
}

func (pg *PGClient) Close() {
	pg.pool.Close()
}
