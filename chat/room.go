package chat

type Room struct {
	Id       int
	Name     string
	Max      int
	Secret   bool
	Password string `json:"-"` // bcrypt hash, never sent to clients
	Admin    string // loginid of the room owner
	Users    map[string]*User
	Banned   map[string]bool `json:"-"` // banned loginids
}

func (c *Room) HasPassword() bool {
	return c.Password != ""
}

func (c *Room) IsAdmin(user *User) bool {
	return user != nil && c.Admin != "" && c.Admin == user.Loginid
}

func (c *Room) GetUsers() map[string]*User {
	return c.Users
}

func (c *Room) GetUsersCount() int {
	return len(c.Users)
}

func (c *Room) Join(user *User) bool {
	c.Users[user.Id] = user

	return true
}

func (c *Room) Exit(user *User) bool {
	delete(c.Users, user.Id)

	return true
}
