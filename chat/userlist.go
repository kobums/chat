package chat

import socketio "github.com/googollee/go-socket.io"

type Userlist struct {
	Items map[string]*User
}

var _users Userlist

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
	user.Room = 0
	user.Socket = so
	c.Items[id] = &user

	return &user
}

func (c *Userlist) Remove(id string) {
	if _, ok := c.Items[id]; ok {
		delete(c.Items, id)
	}
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
