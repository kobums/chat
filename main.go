package main

import (
	"chat/chat"
	"fmt"
	"log"
	"net/http"

	socketio "github.com/googollee/go-socket.io"
	"github.com/gorilla/context"
)

func main() {
	chat := chat.NewChat()
	server := socketio.NewServer(nil)

	server.OnConnect("/", func(s socketio.Conn) error {
		log.Println("connected:", s.ID())
		s.Join("chat")

		return nil
	})

	server.OnEvent("/", "login", func(s socketio.Conn, msg string) string {
		log.Println("login", msg)
		return chat.Login(s, msg)
	})

	server.OnEvent("/", "listroom", func(s socketio.Conn, msg string) string {
		log.Println("listroom", msg)
		return chat.ListRoom(s, msg)
	})

	server.OnEvent("/", "joinroom", func(s socketio.Conn, msg string) {
		log.Println("joinroom", msg)
		chat.JoinRoom(s, msg)
	})

	server.OnEvent("/", "exitroom", func(s socketio.Conn, msg string) {
		log.Println("exitroom", msg)
		chat.ExitRoom(s, msg)
	})

	server.OnEvent("/", "makeroom", func(s socketio.Conn, msg string) {
		log.Println("makeroom", msg)
		chat.MakeRoom(s, msg)
	})

	server.OnEvent("/", "editroom", func(s socketio.Conn, msg string) {
		log.Println("editroom", msg)
		chat.EditRoom(s, msg)
	})

	server.OnEvent("/", "ban", func(s socketio.Conn, msg string) {
		log.Println("ban", msg)
		chat.Ban(s, msg)
	})

	server.OnEvent("/", "kick", func(s socketio.Conn, msg string) {
		log.Println("kick", msg)
		chat.Kick(s, msg)
	})

	server.OnEvent("/", "chat", func(s socketio.Conn, msg string) {
		log.Println("chat", msg)
		chat.Chat(s, msg)
	})

	server.OnEvent("/", "chatto", func(s socketio.Conn, msg string) {
		log.Println("chatto", msg)
		chat.ChatTo(s, msg)
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		fmt.Println("meet error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("disconnection")
		chat.Disconnect(s)
	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving at localhost:9000...")
	log.Fatal(http.ListenAndServe(":9000", context.ClearHandler(http.DefaultServeMux)))
}
