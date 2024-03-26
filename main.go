package main

import (
	"fmt"
	"net/http"
	"os"

	"golang.org/x/net/websocket"
)

var (
	connections = make(map[string]*websocket.Conn)
)

type message struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type allUsersMessage struct {
	Type  string   `json:"type"`
	Users []string `json:"users"`
}

func sendNewUserToOthers(name string) {
	for user, conn := range connections {
		if user == name {
			continue
		}

		websocket.JSON.Send(conn, message{
			Type:    "new_user",
			Message: name,
		})
	}
}

func sendAllUsers(ws *websocket.Conn) {
	allUsers := []string{}
	for user := range connections {
		allUsers = append(allUsers, user)
	}

	websocket.JSON.Send(ws, allUsersMessage{
		Type:  "just_connected",
		Users: allUsers,
	})
}

func broadcastUserDisconnected(name string) {
	allUsers := []string{}
	for user := range connections {
		if user != name {
			allUsers = append(allUsers, user)
		}
	}

	for user, conn := range connections {
		if user == name {
			continue
		}

		websocket.JSON.Send(conn, allUsersMessage{
			Type:  "user_disconnected",
			Users: allUsers,
		})
	}
}

func broadcastMessage(user string, data []byte) {
	for _, conn := range connections {
		websocket.JSON.Send(conn, message{
			Type:    "message",
			Message: fmt.Sprintf("%s: %s", user, string(data)),
		})
	}
}

func handleWS(ws *websocket.Conn) {
	name := ws.Request().URL.Query().Get("name")
	connections[name] = ws

	sendAllUsers(ws)
	sendNewUserToOthers(name)
	broadcastMessage("admin", []byte(fmt.Sprintf("%s connected", name)))

	for {
		var data = make([]byte, 512)
		if _, err := ws.Read(data); err != nil {
			delete(connections, name)
			broadcastUserDisconnected(name)
			broadcastMessage("admin", []byte(fmt.Sprintf("%s disconnected", name)))
			ws.Close()
			break
		}
		broadcastMessage(name, data)
	}
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile("index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(data)
	})

	mux.Handle("GET /ws", websocket.Handler(handleWS))

	err := http.ListenAndServe(":5000", mux)
	if err != nil {
		panic(err)
	}
}
