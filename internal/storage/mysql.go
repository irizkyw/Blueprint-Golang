package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func ConnectMySQL(host, port, user, password, dbName string, timeout time.Duration) (*sql.DB, func(), error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&timeout=%s", user, password, host, port, dbName, timeout.String())

	database, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, nil, err
	}

	if err := database.Ping(); err != nil {
		database.Close()
		return nil, nil, err
	}

	fmt.Println("Connected to MySQL!")
	db = database

	cleanup := func() {
		fmt.Println("Closing MySQL connection...")
		db.Close()
	}

	return db, cleanup, nil
}

func GetDB() *sql.DB {
	return db
}
