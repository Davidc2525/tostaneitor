package main

import (
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
)

// Mark represents a specific point in time during a roasting session, usually to mark an event.
type Mark struct {
	SessionId string  `json:"session_id,omitempty"` // The ID of the session this mark belongs to.
	MarkName  string  `json:"mark_name"`            // The name of the mark (e.g., "First Crack").
	CreatedAt int64   `json:"create_at"`            // The timestamp when the mark was created (in milliseconds).
	OnTemp    float64 `json:"on_temp"`              // The temperature at which the mark was made.
}

// SessionData represents the data of a roasting session that is stored in the database.
type SessionData struct {
	Id       string `json:"id"`        // The unique ID of the session.
	Name     string `json:"name"`      // The name of the session.
	CreateAt int64  `json:"create_at"` // The timestamp when the session was created (in milliseconds).
	EndAt    int64  `json:"end_at"`    // The timestamp when the session ended (in milliseconds).
}

// Session represents an active roasting session.
type Session struct {
	id        string // The unique ID of the session.
	active    bool   // Whether the session is currently active.
	name      string // The name of the session.
	create_at int64  // The timestamp when the session was created (in milliseconds).
}

// NewSession creates a new, inactive session.
func NewSession() *Session {

	return &Session{id: "", active: false}

}

// IsActive returns true if the session is currently active.
func (t Session) IsActive() bool { return t.active }

// GetName returns the name of the session.
func (t Session) GetName() string { return t.name }

// GetId returns the ID of the session.
func (t Session) GetId() string { return t.id }

// GetCreatedAt returns the creation timestamp of the session.
func (t Session) GetCreatedAt() int64 { return t.create_at }

// Start begins a new roasting session.
func (t *Session) Start(name string) error {
	if t.active {
		log.Printf("ya existe una sesion de tostado iniciada: %s\n", t.name)
		return errors.New("ya esta iniciada la session.")
	}
	t.active = true
	t.name = name
	t.id = uuid.NewString()
	t.create_at = time.Now().UnixMilli()
	log.Printf("nueva session %s %s\n", t.id, t.name)

	return nil
}

// Stop ends the current roasting session.
func (t *Session) Stop() {
	if !t.active {
		log.Println("no hay session que detener.")
	} else {
		log.Printf("session %s terminada\n", t.name)
		*t = *NewSession()
	}

}
