
package prediction_internal

import (
	"errors"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type predictorDB *sql.DB

func (db predictorDB) closeDB() error {
	return db.Close()
}
func (db predictorDB) get(k []byte) ([]byte, []byte, bool) {
	var blocks []byte
	var code   []byte
	row := db.QueryRow("SELECT b, c FROM pct WHERE h = ?", k)
	err := row.Scan(&blocks, &code)
	if err != nil {
		return nil, nil, false
	}
	return blocks, code, true
}

func openPredictorDB() (predictorDB, error) {
	return openSqlite("predictors.sqlite3")
}

func openSqlite(fileName string) (*sql.DB, error) {
	//
	db,   err := sql.Open("sqlite3", "file:" + fileName + "?_journal=WAL&_busy_timeout=10&mode=ro")
	if err != nil { return nil, err }
	//
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='P'")
	if err != nil {
		db.Close()
		return nil, err
	}
	if !rows.Next() {
		rows.Close()
		db.Close()
		return nil, errors.New("DB doesn't contain table 'P'")
	}
	rows.Close()
	//
	return db, err
}
