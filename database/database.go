package database

import (
	"context"
	"errors"
	"fmt"
	"log"

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

type DatabaseConfig struct {
	Engine   string
	Host     string
	Port     int
	Name     string
	User     string
	Password string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
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
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	err := rdb.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return rdb, nil
}
