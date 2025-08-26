package main

import (
	"container/list"
	"encoding/json"
	"flag"
	"io"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Davidc2525/go_try/try"
	"github.com/gorilla/websocket"
	ws_client "golang.org/x/net/websocket"
)

// Global variables
var clients = make(map[*websocket.Conn]bool)                        // A map of connected WebSocket clients.
var mu sync.RWMutex                                                 // A mutex to protect access to the clients map.
var current_data = TempType{Type: "temp"}                           // The current temperature data.
var db_temp = list.New()                                            // A list to store temperature data (deprecated).
var session = NewSession()                                          // The current roasting session.
var session_data_provider = NewSessionDataProvider(NewConnection()) // The data provider for session data.

// TempType represents the structure of the temperature data sent over WebSocket.
type TempType struct {
	Type      string  `json:"type"` // The type of the data (e.g., "temp").
	Temp      float64 `json:"temp"` // The temperature value.
	TimeStamp int64   `json:"timestamp"`
	Unit      string  `json:"unit,omitempty"` // The unit of the temperature (e.g., "C").
}

// upgrader is used to upgrade HTTP connections to WebSocket connections.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}

// enabeCORS enables Cross-Origin Resource Sharing (CORS) for the given response writer.
func enabeCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-type")
}

// roastDeleteSessionByIdHandler handles the deletion of a roasting session by its ID.
func roastDeleteSessionByIdHandler(w http.ResponseWriter, r *http.Request) {
	enabeCORS(w)
	session_id := r.PathValue("id")

	response := try.TryArgs[string, map[string]any](
		session_id,
		func(session_id string) (map[string]any, error) {
			session_data_provider.DeleteSession(session_id)
			return map[string]interface{}{"status": true, "msg": "session eliminada"}, nil
		},
		func(e error, session_id string) map[string]any {
			return map[string]interface{}{"status": false, "msg": "error al eliminar session:" + session_id}
		},
	)

	//response := map[string]interface{}{"status": true, "msg": "session eliminada"}

	json_reponse, err := json.Marshal(response)

	if err != nil {
		log.Println(err)
	}

	w.Write(json_reponse)
}

// roastSessionSetMark handles the setting of a mark for a roasting session.
func roastSessionSetMark(w http.ResponseWriter, r *http.Request) {
	enabeCORS(w)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "error reading request body", http.StatusInternalServerError)
	}
	defer r.Body.Close()

	var data Mark

	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "error Unmarshal body", http.StatusInternalServerError)
	}

	log.Println("data mark: ", data)
	session_data_provider.SetMark(data)

}

// roastSessionDataByIdHandler handles the retrieval of data for a roasting session by its ID.
func roastSessionDataByIdHandler(w http.ResponseWriter, r *http.Request) {
	enabeCORS(w)

	data := map[string]interface{}{}

	session_id := r.PathValue("id")

	temps := session_data_provider.GetAllBySessionId(session_id)
	marks := session_data_provider.GetMarksOfSessions(session_id)
	data["temps"] = temps
	data["marks"] = marks

	d, err := json.Marshal(data)

	if err != nil {
		log.Println(err)

	}
	w.Write(d)

}

// roastSessionsHandler handles the retrieval of all roasting sessions.
func roastSessionsHandler(w http.ResponseWriter, r *http.Request) {
	enabeCORS(w)

	data := map[string]interface{}{}

	data["sessions"] = session_data_provider.GetSessions()

	d, err := json.Marshal(data)

	if err != nil {
		log.Println(err)

	}
	w.Write(d)

}

// wsHandler handles WebSocket connections.
func wsHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar la conexión: %v", err)
		return
	}

	// Add the new connection to the map of clients.
	mu.Lock()
	clients[conn] = true
	mu.Unlock()

	log.Printf("Cliente conectado desde: %s. Clientes activos: %d", conn.RemoteAddr(), len(clients))

	// Ensure the connection is removed when it's closed.
	defer func() {
		mu.Lock()
		delete(clients, conn)
		mu.Unlock()
		conn.Close()
		log.Printf("Cliente desconectado de: %s. Clientes activos: %d", conn.RemoteAddr(), len(clients))
	}()

	// Main loop to read messages from the client.
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Cliente desconectado de %s: %v", conn.RemoteAddr(), err)
			break
		}
		var result map[string]interface{}
		err = json.Unmarshal(message, &result)
		if err != nil {
			log.Printf("Error al deserializar JSON: %v", err)
		}
		log.Println("map: ", result)
		log.Printf("Recibido de %s: %s\n", conn.RemoteAddr(), string(message))

		var cmd string
		var exists bool

		cmd, exists = result["cmd"].(string)

		if exists {
			switch cmd {
			case "start":
				log.Println("iniciar session de tostado")
				session.Start(result["session_name"].(string))

				data_respose := map[string]interface{}{"type": "start_response", "msg": "session iniciada"}

				err := session_data_provider.StartNewSession(session.GetId(), session.GetName())

				if err != nil {
					data_respose["error"] = true
					data_respose["msg"] = err.Error()

				} else {
					data_respose["session_id"] = session.GetId()
					data_respose["session_name"] = session.GetName()
				}

				jsonData_response, err := json.Marshal(data_respose)
				if err == nil {

					err := conn.WriteMessage(websocket.TextMessage, []byte(string(jsonData_response)))
					if err != nil {
						log.Printf("Error al enviar a %s: %v", conn.RemoteAddr(), err)
					}
				}

			case "stop":
				log.Println("detener session de tostado")

				session_data_provider.StopSession(session.GetId())
				session.Stop()

			case "get":
				log.Println("obtener info de la sesion acutal si la hay")

				if session.IsActive() {
					log.Println("si hay session")

					d := session_data_provider.GetAllBySessionId(session.GetId())
					marks := session_data_provider.GetMarksOfSessions(session.GetId())

					data_respose := map[string]interface{}{
						"type":               "get_response",
						"msg":                "datos de la session",
						"error":              false,
						"has_session":        true,
						"session_name":       session.GetName(),
						"session_id":         session.GetId(),
						"session_created_at": session.GetCreatedAt(),
					}

					data_respose["temps"] = d
					data_respose["marks"] = marks
					//log.Println("enviando datos de temperatura: ", data_respose)

					jsonData_response, err := json.Marshal(data_respose)

					if err == nil {

						err := conn.WriteMessage(websocket.TextMessage, []byte(string(jsonData_response)))

						if err != nil {
							log.Printf("Error al enviar a %s: %v", conn.RemoteAddr(), err)
						}
					} else {
						log.Printf("Error json a %s\n", err)
					}
				} else {

					data_respose := map[string]interface{}{
						"type":        "get_response",
						"msg":         "no hay session de tostado iniciada",
						"error":       true,
						"has_session": false,
					}
					jsonData_response, err := json.Marshal(data_respose)
					if err == nil {

						err := conn.WriteMessage(websocket.TextMessage, []byte(string(jsonData_response)))
						if err != nil {
							log.Printf("Error al enviar a %s: %v", conn.RemoteAddr(), err)
						}
					} else {
						log.Printf("Error json a %s\n", err)
					}

				}

			default:
				log.Println(cmd, " no es un comando valido")
				return
			}

		}

	}

	log.Printf("Client connected from: %s", conn.RemoteAddr())

}

func main() {

	// Prepare the database.
	session_data_provider.Prepare()

	// Parse command-line flags.
	simule_data := flag.String("s", "false", "si no hay sensor disponible, simular datos de temperatura.")
	host := flag.String("host", "192.168.100.9:81", "Host en el que el servidor escuchará.")
	flag.Parse()
	var ws *ws_client.Conn
	// Connect to the WebSocket server (ESP32).
	u := url.URL{Scheme: "ws", Host: *host, Path: "/"}
	log.Printf("connecting to %s", u.String())
	done := make(chan struct{})

	if *simule_data == "false" {
		ws, err := ws_client.Dial(u.String(), "", "http://localhost/")
		if err != nil {
			log.Println("dial:", err)
			return
		}
		defer ws.Close()

		// Goroutine to receive temperature data from the ESP32.
		go func() {
			defer close(done)
			for {
				var msg string
				err := ws_client.Message.Receive(ws, &msg)

				var temp TempType
				err = json.Unmarshal([]byte(msg), &temp)
				temp.TimeStamp = time.Now().UnixMilli()

				current_data.TimeStamp = temp.TimeStamp
				current_data.Temp = temp.Temp

				go send_data_to_clients()

				if session.IsActive() {
					go session_data_provider.InsertTempValToSession(session.GetId(), temp)
				}

				if err != nil {
					log.Println("read:", err)
					return
				}
				log.Printf("received: %s", msg)
			}
		}()
	} else {
		//datos simulado
		go func() {
			x := 0.0
			for {
				x = x + 0.1
				var temp TempType
				temp.Temp = ((100 * math.Cos(x)) + 30.0) + (rand.Float64() * 10)
				temp.TimeStamp = time.Now().UnixMilli()

				current_data.Temp = temp.Temp
				current_data.TimeStamp = temp.TimeStamp

				go send_data_to_clients()
				if session.IsActive() {
					go session_data_provider.InsertTempValToSession(session.GetId(), temp)
				}

				log.Printf("received rand: %.2f", temp.Temp)

				time.Sleep(1 * time.Second)
			}
		}()
	}

	// Goroutine to send temperature data to the web clients.
	go func() {
		return
		for {
			select {
			case <-time.After(time.Second):
				mu.RLock()
				for clientConn := range clients {
					jsonData, err := json.Marshal(current_data)
					if err == nil {
						err := clientConn.WriteMessage(websocket.TextMessage, []byte(string(jsonData)))
						if err != nil {
							log.Printf("Error al hacer broadcast a %s: %v", clientConn.RemoteAddr(), err)
						}

					}
				}
				mu.RUnlock()
			case <-done:
				return
			}
		}
	}()

	// Goroutine to start the HTTP server.
	go func() {

		fs := http.FileServer(http.Dir("static"))
		mux := http.NewServeMux()
		// Register the WebSocket handler for the "/temp" path.
		mux.HandleFunc("/temp", wsHandler)
		// Register the REST API handlers.
		mux.HandleFunc("/api/v1/temp/roast_sessions", roastSessionsHandler)
		mux.HandleFunc("/api/v1/temp/roast_sessions/{id}", roastSessionDataByIdHandler)
		mux.HandleFunc("DELETE /api/v1/temp/roast_sessions/{id}", roastDeleteSessionByIdHandler)
		mux.HandleFunc("POST /api/v1/temp/roast_sessions/mark", roastSessionSetMark)
		// Register the file server for the root path.
		mux.Handle("/", fs)

		// Start a simple HTTP server to serve the WebSocket endpoint.
		port := ":8080"
		log.Printf("WebSocket server starting on port %s", port)
		err := http.ListenAndServe(port, mux)
		if err != nil {
			log.Printf("Server failed to start: %v", err)
		}
	}()

	// Handle system interrupts (e.g., Ctrl+C).
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	select {
	case <-done:
	case <-interrupt:
		log.Println("interrupt")
		if session.IsActive() {
			session_data_provider.StopSession(session.GetId())
		}
		// Cleanly close the connection by sending a close message to the server, then exit.
		if *simule_data == "false" {
			err := ws.Close()
			if err != nil {
				log.Println("close:", err)
			}
		}

	}
	log.Println("exiting")
}

func send_data_to_clients() {
	mu.RLock()
	for clientConn := range clients {
		jsonData, err := json.Marshal(current_data)
		if err == nil {
			err := clientConn.WriteMessage(websocket.TextMessage, []byte(string(jsonData)))
			if err != nil {
				log.Printf("Error al hacer broadcast a %s: %v", clientConn.RemoteAddr(), err)
			}

		}
	}
	mu.RUnlock()
}
