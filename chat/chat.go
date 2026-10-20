package chat

import (
	"encoding/json"
	"log"

	socketio "github.com/googollee/go-socket.io"
)

type Option struct {
	Size  int    `json:"size"`
	Color string `json:"color"`
}

type Request struct {
	Id       string `json:"id"`
	Nickname string `json:"nickname"`
	Room     int    `json:"room"`
	Password string `json:"password"` // joinroom: room password (if the room has one)
}

type LoginRequest struct {
	Loginid  string `json:"loginid"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

type MessageRequest struct {
	Nickname string `json:"nickname"`
	Room     int    `json:"room"`
	Text     string `json:"text"`
	Option   Option `json:"option"`
}

type MessageToRequest struct {
	Nickname string `json:"nickname"`
	Room     int    `json:"room"`
	Text     string `json:"text"`
	To       string `json:"to"`
	Option   Option `json:"option"`
}

type MakeRoomRequest struct {
	Name     string `json:"name"`
	Max      int    `json:"max"`
	Secret   bool   `json:"secret"`
	Password string `json:"password"`
}

type Response struct {
	Id      string `json:"id"`
	Status  int    `json:"status"`
	Message string `json:"message"`
	Room    int    `json:"room"`
}

type MessageResponse struct {
	Id       string `json:"id"`
	Nickname string `json:"nickname"`
	Room     int    `json:"room"`
	Text     string `json:"text"`
	Option   Option `json:"option"`
}

type Chat struct {
	Userlist *Userlist
	Roomlist *Roomlist
	Accounts *Accounts
}

func NewChat() *Chat {
	var item Chat
	item.Userlist = NewUserlist()
	item.Accounts = NewAccounts()

	item.Roomlist = NewRoomlist()
	item.Roomlist.Add("Wait Room", 100000, false, "", "")

	return &item
}

func (c *Chat) MakeMessage(id string, nickname string, text string, option *Option) string {
	var message MessageResponse
	message.Id = id
	message.Nickname = nickname
	message.Text = text
	if option != nil {
		message.Option = *option
	} else {
		message.Option.Size = 0
		message.Option.Color = ""
	}

	ret, _ := json.Marshal(message)
	return string(ret)
}

func (c *Chat) Broadcating(roomid int, code string, message string) {
	room := c.Roomlist.Find(roomid)
	if room == nil {
		return
	}

	users := room.GetUsers()

	for _, user := range users {
		user.Socket.Emit(code, message)
	}
}

func (c *Chat) BroadcatingWithout(id string, roomid int, code string, message string) {
	room := c.Roomlist.Find(roomid)
	if room == nil {
		return
	}

	users := room.GetUsers()

	for _, user := range users {
		if user.Id == id {
			continue
		}

		user.Socket.Emit(code, message)
	}
}

func (c *Chat) FindUser(so socketio.Conn) *User {
	id := so.ID()

	item := c.Userlist.Find(id)

	return item
}

func (c *Chat) Login(so socketio.Conn, msg string) string {
	id := so.ID()
	var response Response
	response.Id = id
	response.Room = 0

	var item LoginRequest
	err := json.Unmarshal([]byte(msg), &item)
	if err != nil {
		response.Status = 0
		response.Message = "아이디를 찾을수가 없습니다"
	} else if !c.Accounts.Verify(item.Loginid, item.Password) {
		response.Status = 0
		response.Message = "아이디 또는 비밀번호가 올바르지 않습니다"
		log.Println("login failed:", item.Loginid)
	} else {
		response.Status = 1
		log.Println("login:", item.Loginid)

		find := c.Userlist.FindByLoginid(item.Loginid)

		if find == nil {
			find = c.Userlist.Add(id, item.Loginid, item.Nickname, so)
		} else {
			response.Room = find.Room
		}

		room := c.Roomlist.Find(response.Room)
		if room != nil && room.Join(find) == true {
			c.BroadcatingWithout(id, response.Room, "joinroom", c.MakeMessage(id, item.Nickname, "", nil))
		}
	}

	ret, _ := json.Marshal(response)
	return string(ret)
}

func (c *Chat) Disconnect(so socketio.Conn) {
	id := so.ID()
	item := c.Userlist.Find(id)

	if item != nil {
		log.Printf("room = %v\n", item.Room)
		room := c.Roomlist.Find(item.Room)
		if room != nil {
			room.Exit(item)
			c.Broadcating(item.Room, "exitroom", c.MakeMessage(id, item.Nickname, "", nil))

			if room.Id != 0 && room.GetUsersCount() == 0 {
				c.Roomlist.Remove(room.Id)
			}
		}

		c.Userlist.Remove(id)
	}
}

func Error(so socketio.Conn, err error) {
	log.Println("error:", err)
}

func (c *Chat) JoinRoom(so socketio.Conn, msg string) string {
	id := so.ID()
	item := c.Userlist.Find(id)

	if item == nil {
		return ""
	}

	var req Request
	err := json.Unmarshal([]byte(msg), &req)

	if err != nil {
		return ""
	}

	if item.Room == req.Room {
		return ""
	}

	target := c.Roomlist.Find(req.Room)
	if target == nil {
		return ""
	}

	if target.Banned[item.Loginid] {
		so.Emit("joinroom_fail", c.MakeMessage(id, item.Nickname, "강퇴된 방입니다", nil))
		return ""
	}

	if target.HasPassword() && !target.IsAdmin(item) && !checkPassword(target.Password, req.Password) {
		so.Emit("joinroom_fail", c.MakeMessage(id, item.Nickname, "비밀번호가 올바르지 않습니다", nil))
		return ""
	}

	room := c.Roomlist.Find(item.Room)
	if room != nil {
		room.Exit(item)
		c.Broadcating(item.Room, "exitroom", c.MakeMessage(id, item.Nickname, "", nil))
	}

	room = target
	if room.Join(item) == true {
		c.Broadcating(req.Room, "joinroom", c.MakeMessage(id, item.Nickname, "", nil))
	}

	return ""
}

func (c *Chat) ExitRoom(so socketio.Conn, msg string) string {
	id := so.ID()
	item := c.Userlist.Find(id)

	if item == nil {
		return ""
	}

	var req Request
	err := json.Unmarshal([]byte(msg), &req)

	if err != nil {
		return ""
	}

	if item.Room == req.Room {
		return ""
	}

	room := c.Roomlist.Find(req.Room)
	if room != nil {
		room.Exit(item)
		c.Broadcating(req.Room, "exitroom", c.MakeMessage(id, item.Nickname, "", nil))
	}

	room = c.Roomlist.Find(0)
	if room != nil && room.Join(item) == true {
		c.Broadcating(0, "joinroom", c.MakeMessage(id, item.Nickname, "", nil))
	}

	return ""
}

func (c *Chat) ListRoom(so socketio.Conn, msg string) string {
	rooms := c.Roomlist.GetList()
	e, err := json.Marshal(rooms)
	if err != nil {
		return ""
	}

	return string(e)
}

func (c *Chat) MakeRoom(so socketio.Conn, msg string) string {
	id := so.ID()
	item := c.Userlist.Find(id)

	if item == nil {
		return ""
	}

	var req MakeRoomRequest
	err := json.Unmarshal([]byte(msg), &req)

	if err != nil {
		return ""
	}

	password, err := hashPassword(req.Password)
	if err != nil {
		return ""
	}

	roomId := c.Roomlist.Add(req.Name, req.Max, req.Secret, password, item.Loginid)

	room := c.Roomlist.Find(item.Room)
	if room != nil {
		room.Exit(item)
		c.Broadcating(item.Room, "exitroom", c.MakeMessage(id, item.Nickname, "", nil))
	}

	room = c.Roomlist.Find(roomId)
	if room != nil && room.Join(item) == true {
	}

	return ""
}

// EditRoom is not implemented yet. When it is, it must be restricted to the
// room owner (room.IsAdmin) like Ban/Kick.
func (c *Chat) EditRoom(so socketio.Conn, msg string) string {
	return ""
}

// removeFromRoom handles ban/kick: only the owner of the room may remove
// another user, and the removed user is sent back to the wait room.
// req.Id is the target user's socket id (the "id" field of chat messages).
func (c *Chat) removeFromRoom(so socketio.Conn, msg string, ban bool) string {
	id := so.ID()
	item := c.Userlist.Find(id)

	if item == nil {
		return ""
	}

	var req Request
	err := json.Unmarshal([]byte(msg), &req)

	if err != nil {
		return ""
	}

	room := c.Roomlist.Find(req.Room)
	if room == nil || room.Id == 0 || !room.IsAdmin(item) {
		log.Println("ban/kick refused: not room owner", id)
		return ""
	}

	target := c.Userlist.Find(req.Id)
	if target == nil || target.Id == id || room.Users[target.Id] == nil {
		return ""
	}

	if ban {
		room.Banned[target.Loginid] = true
	}

	room.Exit(target)
	c.Broadcating(room.Id, "exitroom", c.MakeMessage(target.Id, target.Nickname, "", nil))
	if ban {
		target.Socket.Emit("ban", c.MakeMessage(target.Id, target.Nickname, "", nil))
	} else {
		target.Socket.Emit("kick", c.MakeMessage(target.Id, target.Nickname, "", nil))
	}

	wait := c.Roomlist.Find(0)
	if wait != nil && wait.Join(target) == true {
		c.Broadcating(0, "joinroom", c.MakeMessage(target.Id, target.Nickname, "", nil))
	}

	return ""
}

func (c *Chat) Ban(so socketio.Conn, msg string) string {
	return c.removeFromRoom(so, msg, true)
}

func (c *Chat) Kick(so socketio.Conn, msg string) string {
	return c.removeFromRoom(so, msg, false)
}

func (c *Chat) Chat(so socketio.Conn, msg string) string {
	id := so.ID()
	item := c.Userlist.Find(id)

	if item == nil {
		return ""
	}

	var req MessageRequest
	err := json.Unmarshal([]byte(msg), &req)

	if err != nil {
		return ""
	}

	c.Broadcating(item.Room, "chat", c.MakeMessage(id, item.Nickname, req.Text, &req.Option))

	return ""
}

func (c *Chat) ChatTo(so socketio.Conn, msg string) string {
	id := so.ID()
	item := c.Userlist.Find(id)

	if item == nil {
		return ""
	}

	var req MessageToRequest
	err := json.Unmarshal([]byte(msg), &req)

	if err != nil {
		return ""
	}

	room := c.Roomlist.Find(req.Room)
	if room != nil {
		users := room.GetUsers()

		if to := users[req.To]; to != nil {
			to.Socket.Emit("chatto", c.MakeMessage(id, item.Nickname, req.Text, &req.Option))
		}
	}

	return ""
}
