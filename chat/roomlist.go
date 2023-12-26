package chat

type Roomlist struct {
	Items map[int]Room
	Id    int
}

func NewRoomlist() *Roomlist {
	var item Roomlist

	item.Id = 0
	item.Items = make(map[int]Room)

	return &item
}

func (c *Roomlist) GetList() *map[int]Room {
	return &c.Items
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
	c.Items[id] = room

	return id
}

func (c *Roomlist) Remove(id int) {
	if _, ok := c.Items[id]; ok {
		delete(c.Items, id)
	}
}

func (c *Roomlist) Find(id int) *Room {
	if v, ok := c.Items[id]; ok {
		return &v
	}

	return nil
}
