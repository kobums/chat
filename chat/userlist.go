package chat

import socketio "github.com/googollee/go-socket.io"

// Userlist is not safe for concurrent use on its own; every access goes
// through Chat, which serializes it with Chat.mu.
type Userlist struct {
	Items map[string]*User
}

func NewUserlist() *Userlist {
	var userlist Userlist
	userlist.Items = make(map[string]*User)

	return &userlist
}

func (c *Userlist) Add(id string, loginid string, nickname string, so socketio.Conn) *User {
	if item, ok := c.Items[id]; ok {
		return item
	}

	var user User
	user.Id = id
	user.Loginid = loginid
	user.Nickname = nickname
	user.Status = WAIT
	user.Room = WAIT_ROOM
	user.Socket = so
	c.Items[id] = &user

	return &user
}

func (c *Userlist) Remove(id string) {
	delete(c.Items, id)
}

func (c *Userlist) Find(id string) *User {
	if user, ok := c.Items[id]; ok {
		return user
	}

	return nil
}

func (c *Userlist) FindByLoginid(loginid string) *User {
	for _, user := range c.Items {
		if user.Loginid == loginid {
			return user
		}
	}

	return nil
}
