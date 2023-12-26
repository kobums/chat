package chat

type Room struct {
	Id       int
	Name     string
	Max      int
	Secret   bool
	Password string
	Admin    string
	Users    map[string]*User
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
