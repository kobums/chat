package chat

// Roomlist is not safe for concurrent use on its own; every access goes
// through Chat, which serializes it with Chat.mu.
type Roomlist struct {
	Items map[int]*Room
	Id    int
}

func NewRoomlist() *Roomlist {
	var item Roomlist

	item.Id = 0
	item.Items = make(map[int]*Room)

	return &item
}

func (c *Roomlist) GetList() map[int]*Room {
	return c.Items
}

func (c *Roomlist) Add(name string, max int, secret bool, password string, admin string) int {
	id := c.Id
	c.Id++

	var room Room
	room.Id = id
	room.Name = name
	room.Max = max
	room.Secret = secret
	room.Password = password
	room.Admin = admin
	room.Users = make(map[string]*User)
	room.Banned = make(map[string]bool)
	c.Items[id] = &room

	return id
}

func (c *Roomlist) Remove(id int) {
	delete(c.Items, id)
}

func (c *Roomlist) Find(id int) *Room {
	if v, ok := c.Items[id]; ok {
		return v
	}

	return nil
}
