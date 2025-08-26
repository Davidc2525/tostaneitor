package main

import (
	"database/sql"
	"errors"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SessionDataProvider provides an abstraction layer for interacting with the database.
type SessionDataProvider struct {
	Db *sql.DB // The database connection.
}

// NewSessionDataProvider creates a new SessionDataProvider.
func NewSessionDataProvider(db *sql.DB) *SessionDataProvider {
	sdp := &SessionDataProvider{Db: db}
	log.Println("new session data provider created ", sdp)
	return sdp
}

// GetSessions retrieves all roasting sessions from the database.
func (this SessionDataProvider) GetSessions() []SessionData {
	data := []SessionData{}
	get_sql := `
		SELECT session_id,session_name,created_at,end_at FROM sessions 
	`

	rows, err := this.Db.Query(get_sql)
	if err != nil {
		log.Println("error al obtener temps,", err)
		return data
	}

	for rows.Next() {
		var sID string
		var sName string
		var sCat int64
		var sEat int64

		if err := rows.Scan(&sID, &sName, &sCat, &sEat); err != nil {
			log.Println(err)
		}

		session := SessionData{
			Id:       sID,
			Name:     sName,
			CreateAt: sCat,
			EndAt:    sEat,
		}
		data = append(data, session)
	}

	return data
}

// StartNewSession creates a new roasting session in the database.
func (this SessionDataProvider) StartNewSession(session_id string, session_name string) error {

	sql := `
INSERT INTO sessions (session_id,session_name,created_at,end_at)
VALUES (?,?,?,?) 
	`
	current_time := time.Now()

	_, err := this.Db.Exec(sql, session_id, session_name, current_time.UnixMilli(), 0)
	if err != nil {
		log.Println("error al crear session", err)
		return errors.New("error_create_session")
	}
	return nil
}

// StopSession updates the end time of a roasting session in the database.
func (this SessionDataProvider) StopSession(session_id string) {
	log.Println("stop session: ", session_id)
	sql := `
UPDATE sessions set end_at = ?
WHERE session_id = ?
	`
	current_time := time.Now()

	_, err := this.Db.Exec(sql, current_time.UnixMilli(), session_id)
	if err != nil {
		log.Println("error al detener session ", err)
	}
}

// DeleteSession deletes a roasting session and its associated measurements from the database.
func (this SessionDataProvider) DeleteSession(session_id string) {

	sql := `
	DELETE FROM sessions WHERE session_id = ?;
	DELETE FROM measurements WHERE session_id = ?;
	`

	_, err := this.Db.Exec(sql, session_id, session_id)
	if err != nil {
		log.Println(err)
	}
}

// InsertTempValToSession inserts a temperature value for a given session into the database.
func (this SessionDataProvider) InsertTempValToSession(session_id string, temp TempType) {

	insert_sql := `
		INSERT INTO measurements (session_id,timestamp,temp_val)
		VALUES(?,?,?)`

	_, err := this.Db.Exec(insert_sql, session_id, temp.TimeStamp, temp.Temp)
	if err != nil {
		log.Println("error al insertar temp", err)
	}

}

// GetAllBySessionId retrieves all temperature measurements for a given session from the database.
func (this *SessionDataProvider) GetAllBySessionId(session_id string) []*TempType {

	data := []*TempType{}
	get_sql := `
		SELECT * FROM measurements WHERE session_id = ? ORDER BY timestamp 
	`

	rows, err := this.Db.Query(get_sql, session_id)
	if err != nil {
		log.Println("error al obtener temps,", err)
		return data
	}

	for rows.Next() {
		var sID string
		var ts int64
		var temp float64

		if err := rows.Scan(&sID, &ts, &temp); err != nil {
			log.Println(err)
		}

		var temp_ *TempType = &TempType{Temp: temp, TimeStamp: ts}
		data = append(data, temp_)
	}

	return data

}

// SetMark inserts a new mark for a session into the database.
func (this SessionDataProvider) SetMark(mark Mark) {

	if mark.SessionId == "" {
		return
	}
	sql := `

INSERT INTO session_marks (session_id,mark_name,created_at,on_temp) 
VALUES (?,?,?,?);
`

	_, err := this.Db.Exec(sql, mark.SessionId, mark.MarkName, mark.CreatedAt, mark.OnTemp)
	if err != nil {
		log.Println("error al set mark", err)
	}
}

// GetMarksOfSessions retrieves all marks for a given session from the database.
func (this SessionDataProvider) GetMarksOfSessions(session_id string) []Mark {
	marks := []Mark{}
	get_sql := `
		SELECT mark_name,created_at,on_temp from session_marks where session_id = ?
	`

	rows, err := this.Db.Query(get_sql, session_id)
	if err != nil {
		log.Println("error al obtener temps,", err)
		return marks
	}

	for rows.Next() {
		var markName string
		var markCat int64
		var markOnTemp float64

		if err := rows.Scan(&markName, &markCat, &markOnTemp); err != nil {
			log.Println(err)
		}

		mark := Mark{
			MarkName:  markName,
			CreatedAt: markCat,
			OnTemp:    markOnTemp,
		}
		marks = append(marks, mark)
	}

	return marks
}

// Prepare creates the necessary tables in the database if they don't already exist.
func (this SessionDataProvider) Prepare() {

	create_sql := `
create table if NOT EXISTS measurements 
(
	session_id text NOT NULL,
  	timestamp integer not null,
	temp_val real not null,
  	PRIMARY KEY (session_id,timestamp)
);

create table if NOT EXISTS sessions 
(
	session_id text NOT NULL,
  	session_name text not null,
  	created_at integer not null,
	end_at integer not null,
  	PRIMARY KEY (created_at,session_id)
);

create table if NOT EXISTS session_marks
(
	session_id text NOT NULL,
  	mark_name text not null,
  	created_at integer not null,
	on_temp real not null,
  	PRIMARY KEY (session_id,created_at)
);
	`

	_, err := this.Db.Exec(create_sql)
	if err != nil {
		log.Printf("error al crear la tablas")
	}

	log.Println("tablas creadas con exito.")
}
