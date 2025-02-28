package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"              // PostgreSQL driver
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Enum database
type DBType string

const (
	MySQL      DBType = "mysql"
	PostgreSQL DBType = "postgres"
	MongoDB    DBType = "mongodb"
)

type Database struct {
	sqlDB   *sql.DB
	mongoDB *mongo.Client
	dbType  DBType
}

func ConnectDatabase(dbType DBType, host, port, user, password, dbName string, timeout time.Duration) (*Database, func(), error) {
	var db Database
	db.dbType = dbType
	var cleanup func()

	switch dbType {
	case MySQL:
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&timeout=%s", user, password, host, port, dbName, timeout.String())
		sqlDB, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, nil, err
		}
		if err := sqlDB.Ping(); err != nil {
			sqlDB.Close()
			return nil, nil, err
		}
		db.sqlDB = sqlDB
		cleanup = func() {
			fmt.Println("Closing MySQL connection...")
			db.sqlDB.Close()
		}

	case PostgreSQL:
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=%d",
			host, port, user, password, dbName, int(timeout.Seconds()))
		sqlDB, err := sql.Open("postgres", dsn)
		if err != nil {
			return nil, nil, err
		}
		if err := sqlDB.Ping(); err != nil {
			sqlDB.Close()
			return nil, nil, err
		}
		db.sqlDB = sqlDB
		cleanup = func() {
			fmt.Println("Closing PostgreSQL connection...")
			db.sqlDB.Close()
		}

	case MongoDB:
		mongoURI := fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", user, password, host, port, dbName)
		client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
		if err != nil {
			return nil, nil, err
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err = client.Connect(ctx)
		if err != nil {
			return nil, nil, err
		}
		db.mongoDB = client
		cleanup = func() {
			fmt.Println("Closing MongoDB connection...")
			db.mongoDB.Disconnect(context.Background())
		}

	default:
		return nil, nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	fmt.Printf("Connected to %s!\n", dbType)
	return &db, cleanup, nil
}

func (db *Database) GetSQLDB() *sql.DB {
	if db.dbType == MySQL || db.dbType == PostgreSQL {
		return db.sqlDB
	}
	return nil
}

func (db *Database) GetMongoDB() *mongo.Client {
	if db.dbType == MongoDB {
		return db.mongoDB
	}
	return nil
}
