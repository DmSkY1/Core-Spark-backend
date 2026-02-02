package database

import (
	"context"
	"fmt"
	"gobackend/internal/app/models"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func DBConnection() *pgxpool.Pool {

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	dbURL := os.Getenv("DBURL")

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		fmt.Println(err)
	}
	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 10 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		panic(err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to ping database: %v\n", err)
		panic(err)
	}

	return pool
}

func RedisConnection(ctx context.Context, cfg models.Redis_Config_Model) (*redis.Client, error) {
	db := redis.NewClient(&redis.Options{
		Addr:            cfg.Addr,
		Password:        cfg.Password,
		Username:        cfg.User,
		DB:              cfg.DB,
		MaxRetries:      cfg.MaxRetries,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.Timeout,
		WriteTimeout:    cfg.Timeout,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		PoolTimeout:     cfg.PoolTimeout,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
	})

	if err := db.Ping(ctx).Err(); err != nil {
		fmt.Printf("failed to connect to redis server: %s\n", err.Error())
		return nil, err
	}

	return db, nil
}
