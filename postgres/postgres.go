package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/diki-haryadi/go-micro-template/pkg/config"
)

type Config struct {
	Host    string
	Port    string
	User    string
	Pass    string
	DBName  string
	SslMode string
}

type Postgres struct {
	SqlxDB *sqlx.DB
}

func (db *Postgres) Close() {
	_ = db.SqlxDB.DB.Close()
	_ = db.SqlxDB.Close()
}

func NewConnection(ctx context.Context, conf *Config) (*Postgres, error) {
	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		conf.Host,
		conf.Port,
		conf.User,
		conf.Pass,
		conf.DBName,
		conf.SslMode,
	)

	db, err := sqlx.ConnectContext(ctx, "postgres", connString)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(config.BaseConfig.Postgres.MaxConn)                           // the defaultLogger is 0 (unlimited)
	db.SetMaxIdleConns(config.BaseConfig.Postgres.MaxIdleConn)                       // defaultMaxIdleConn = 2
	db.SetConnMaxLifetime(time.Duration(config.BaseConfig.Postgres.MaxLifeTimeConn)) // 0, connections are reused forever

	if err := db.Ping(); err != nil {
		fmt.Println("can not ping postgres")
		defer db.Close()

		return nil, err
	}

	return &Postgres{SqlxDB: db}, nil
}
