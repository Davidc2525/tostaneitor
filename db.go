package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// NewConnection creates and returns a new connection to the SQLite database.
func NewConnection() *sql.DB {

	// Open a connection to the SQLite database file.
	db, err := sql.Open("sqlite3", "./data/data.db")
	if err != nil {
		log.Println(err)
	}
	//defer db.Close()

	log.Println("new sql created")
	return db

}
