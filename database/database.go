package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql" // mysql driver
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var (
	// ErrInvalidDBName database name is missing.
	ErrInvalidDBName = errors.New("database name is missing")

	// ErrInvalidDBUser database user is missing.
	ErrInvalidDBUser = errors.New("database user is missing")
)

type PoolConfig struct {
	MaxLifeTime  int32 `yaml:"maxLifeTime" default:"10"`
	MaxOpenConns int32 `yaml:"maxOpenConns" defaul:"5"`
	MaxIdleConns int32 `yaml:"maxIdleConns" default:"3"`
}

// DatabaseConfig is the configuration for a sql database.
type DatabaseConfig struct {
	Engine   string     `yaml:"engine"`
	Host     string     `yaml:"host"`
	Port     int        `yaml:"port"`
	Name     string     `yaml:"name"`
	User     string     `yaml:"user"`
	Password string     `yaml:"password"`
	Pool     PoolConfig `yaml:"pool"`
}

type SentielConfig struct {
	Enabled    bool     `yaml:"enabled"`
	Addresses  []string `yaml:"addresses"`
	Password   string   `yaml:"password"`
	MasterName string   `yaml:"master_name"`
}

// RedisConfig is the configuration for a redis database.
type RedisConfig struct {
	Addr     string        `yaml:"addr"`
	Password string        `yaml:"password"`
	DB       int           `yaml:"db"`
	Sentinel SentielConfig `yaml:"Sentinel"`
}

// CreateMySqlConnection creates a connection to a mysql database
func CreateMySqlConnection(ctx context.Context, dbConfig DatabaseConfig) (*sqlx.DB, error) {
	var connectionString string
	var db *sqlx.DB
	var err error

	if dbConfig.User == "" {
		return nil, ErrInvalidDBUser
	}

	connectionString = fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true",
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Name)

	log.Println("Connecting to database...")
	db, err = sqlx.ConnectContext(ctx, "mysql", connectionString)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(time.Duration(time.Duration(dbConfig.Pool.MaxLifeTime).Seconds()))
	db.SetMaxOpenConns(int(dbConfig.Pool.MaxOpenConns))
	db.SetMaxIdleConns(int(dbConfig.Pool.MaxIdleConns))

	log.Println("Connected to database")
	return db, nil
}

// CreateSqliteConnection creates a connection to a sqlite database
func CreateSqliteConnection(ctx context.Context, dbConfig DatabaseConfig) (*sqlx.DB, error) {
	if dbConfig.Name == "" {
		return nil, ErrInvalidDBName
	}

	log.Println("Connecting to database...")
	source := fmt.Sprintf("./%s.db", dbConfig.Name)

	db, err := sqlx.ConnectContext(ctx, "sqlite3", source)
	if err != nil {
		return nil, err
	}

	log.Println("Connected to database")
	return db, nil
}

// CreateRedisConnection creates a connection to a redis database
func CreateRedisConnection(ctx context.Context, config RedisConfig) (*redis.Client, error) {
	var rdb *redis.Client

	if config.Sentinel.Enabled {
		rdb = redis.NewFailoverClient(&redis.FailoverOptions{
			SentinelAddrs:    config.Sentinel.Addresses,
			Password:         config.Password,
			SentinelPassword: config.Sentinel.Password,
			DB:               config.DB,
			MasterName:       config.Sentinel.MasterName,
		})
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr:     config.Addr,
			Password: config.Password,
			DB:       config.DB,
		})
	}

	err := rdb.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return rdb, nil
}
