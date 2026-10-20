package chat

import (
	socketio "github.com/googollee/go-socket.io"
)

const (
	WAIT   = 0
	NORMAL = 1
)

// WAIT_ROOM is the id of the lobby every user is in when not in a room.
const WAIT_ROOM = 0

type User struct {
	Id       string
	Loginid  string
	Nickname string
	Status   int
	Room     int
	Socket   socketio.Conn `json:"-"`
}

func (c *User) SetStatus(id string, status int) {
	c.Status = status
}

func (c *User) SetRoom(id string, room int) {
	c.Room = room
}
