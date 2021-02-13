package database

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

var Dbhost string
var Dbname string
var Dbuser string
var Dbpass string

// PerformQuery : Performs a MySQL query over the device's Edgebox API
func PerformQuery() string {

	// Will try to connect to API database, which should be running locally under WS.
	db, err := sql.Open("mysql", Dbuser+":"+Dbpass+"@tcp("+Dbhost+")/"+Dbname)

	// if there is an error opening the connection, handle it
	if err != nil {
		panic(err.Error())
	}

	// defer the close till after the main function has finished executing
	defer db.Close()

	// perform a db.Query insert
	insert, err := db.Query("INSERT INTO options (name, value) VALUES ( 'TEST_OPTION_SYSCTL', 'TEST' );")

	// if there is an error inserting, handle it
	if err != nil {
		panic(err.Error())
	}

	// be careful deferring Queries if you are using transactions
	defer insert.Close()

	return "OK"

}
