package chat

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"

	socketio "github.com/googollee/go-socket.io"
)

const (
	maxRoomUsers   = 1000
	maxRoomNameLen = 100
)

// FlexInt accepts a JSON number or a numeric string ("10"), because the web
// client reads values from text inputs. An empty string or null is 0.
type FlexInt int

func (f *FlexInt) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		*f = 0
		return nil
	}

	if len(data) > 0 && data[0] == '"' {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}

		str = strings.TrimSpace(str)
		if str == "" {
			*f = 0
			return nil
		}

		v, err := strconv.Atoi(str)
		if err != nil {
			return errors.New("invalid number: " + str)
		}

		*f = FlexInt(v)
		return nil
	}

	var v int
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	*f = FlexInt(v)
	return nil
}

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
	Name     string  `json:"name"`
	Max      FlexInt `json:"max"`
	Secret   bool    `json:"secret"`
	Password string  `json:"password"`
}

// EditRoomRequest changes a room's settings. Only the fields that are sent are
// changed; Password "" removes the room password.
type EditRoomRequest struct {
	Room     int      `json:"room"`
	Name     *string  `json:"name"`
	Max      *FlexInt `json:"max"`
	Password *string  `json:"password"`
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

// delivery is one socket emit, queued while Chat.mu is held and sent after
// it is released (Emit blocks until the peer's writer takes the packet, so a
// slow client must not stall every other handler).
type delivery struct {
	so    socketio.Conn
	event string
	msg   string
}

type outbox []delivery

func (o outbox) send() {
	for _, d := range o {
		d.so.Emit(d.event, d.msg)
	}
}

// Chat holds all server state. Socket.io handlers run concurrently, so every
// access to Userlist / Roomlist / Room / User goes through mu.
type Chat struct {
	mu       sync.Mutex
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

func makeResponse(id string, status int, message string, room int) string {
	ret, _ := json.Marshal(Response{Id: id, Status: status, Message: message, Room: room})
	return string(ret)
}

// Broadcating queues message for every user in the room. Caller holds c.mu.
func (c *Chat) Broadcating(out *outbox, roomid int, code string, message string) {
	c.BroadcatingWithout(out, "", roomid, code, message)
}

// BroadcatingWithout queues message for every user in the room except id.
// Caller holds c.mu.
func (c *Chat) BroadcatingWithout(out *outbox, id string, roomid int, code string, message string) {
	room := c.Roomlist.Find(roomid)
	if room == nil {
		return
	}

	for _, user := range room.Users {
		if user.Id == id {
			continue
		}

		*out = append(*out, delivery{user.Socket, code, message})
	}
}

// leaveRoom takes the user out of its current room, tells the others, and
// removes the room when the last user left (except the wait room).
// Caller holds c.mu.
func (c *Chat) leaveRoom(out *outbox, item *User) {
	room := c.Roomlist.Find(item.Room)
	if room == nil {
		return
	}

	room.Exit(item)
	c.Broadcating(out, room.Id, "exitroom", c.MakeMessage(item.Id, item.Nickname, "", nil))

	if room.Id != WAIT_ROOM && room.GetUsersCount() == 0 {
		c.Roomlist.Remove(room.Id)
	}
}

// enterRoom moves the user from its current room into room. Caller holds c.mu.
func (c *Chat) enterRoom(out *outbox, item *User, room *Room) {
	if item.Room != room.Id {
		c.leaveRoom(out, item)
	}

	if room.Join(item) {
		c.Broadcating(out, room.Id, "joinroom", c.MakeMessage(item.Id, item.Nickname, "", nil))
	}
}

func (c *Chat) FindUser(so socketio.Conn) *User {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Userlist.Find(so.ID())
}

func (c *Chat) Login(so socketio.Conn, msg string) string {
	id := so.ID()
	var response Response
	response.Id = id
	response.Room = WAIT_ROOM

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

		var out outbox
		c.mu.Lock()

		roomId := WAIT_ROOM
		user := c.Userlist.Find(id)
		if user == nil {
			// A previous session of the same account (e.g. the client
			// reconnected before the old socket's disconnect arrived) is
			// replaced by this socket; it keeps the room it was in.
			if old := c.Userlist.FindByLoginid(item.Loginid); old != nil {
				roomId = old.Room
				if room := c.Roomlist.Find(old.Room); room != nil {
					room.Exit(old)
				}
				c.Userlist.Remove(old.Id)
			}

			user = c.Userlist.Add(id, item.Loginid, item.Nickname, so)
		} else {
			roomId = user.Room
		}

		room := c.Roomlist.Find(roomId)
		if room == nil || room.Banned[user.Loginid] {
			room = c.Roomlist.Find(WAIT_ROOM)
		}
		if room != nil && room.Join(user) {
			c.BroadcatingWithout(&out, id, room.Id, "joinroom", c.MakeMessage(id, user.Nickname, "", nil))
		}
		response.Room = user.Room

		c.mu.Unlock()
		out.send()
	}

	ret, _ := json.Marshal(response)
	return string(ret)
}

func (c *Chat) Disconnect(so socketio.Conn) {
	id := so.ID()

	var out outbox
	c.mu.Lock()

	item := c.Userlist.Find(id)
	if item != nil {
		c.leaveRoom(&out, item)
		c.Userlist.Remove(id)
	}

	c.mu.Unlock()
	out.send()
}

func Error(so socketio.Conn, err error) {
	log.Println("error:", err)
}

func (c *Chat) JoinRoom(so socketio.Conn, msg string) string {
	id := so.ID()

	var req Request
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		return ""
	}

	var out outbox
	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		out.send()
	}()

	item := c.Userlist.Find(id)
	if item == nil {
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
		out = append(out, delivery{so, "joinroom_fail", c.MakeMessage(id, item.Nickname, "강퇴된 방입니다", nil)})
		return ""
	}

	if target.HasPassword() && !target.IsAdmin(item) && !checkPassword(target.Password, req.Password) {
		out = append(out, delivery{so, "joinroom_fail", c.MakeMessage(id, item.Nickname, "비밀번호가 올바르지 않습니다", nil)})
		return ""
	}

	if target.IsFull() && !target.IsAdmin(item) {
		out = append(out, delivery{so, "joinroom_fail", c.MakeMessage(id, item.Nickname, "방이 가득 찼습니다", nil)})
		return ""
	}

	c.enterRoom(&out, item, target)

	return ""
}

// ExitRoom leaves the room the user is currently in (req.Room must be that
// room) and goes back to the wait room.
func (c *Chat) ExitRoom(so socketio.Conn, msg string) string {
	id := so.ID()

	var req Request
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		return ""
	}

	var out outbox
	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		out.send()
	}()

	item := c.Userlist.Find(id)
	if item == nil {
		return ""
	}

	if item.Room != req.Room || item.Room == WAIT_ROOM {
		return ""
	}

	wait := c.Roomlist.Find(WAIT_ROOM)
	if wait != nil {
		c.enterRoom(&out, item, wait)
	}

	return ""
}

func (c *Chat) ListRoom(so socketio.Conn, msg string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, err := json.Marshal(c.Roomlist.GetList())
	if err != nil {
		return ""
	}

	return string(e)
}

func validateRoomName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	return name, name != "" && len([]rune(name)) <= maxRoomNameLen
}

func validateRoomMax(max int) bool {
	return max > 0 && max <= maxRoomUsers
}

func (c *Chat) MakeRoom(so socketio.Conn, msg string) string {
	id := so.ID()

	var req MakeRoomRequest
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		return makeResponse(id, 0, "잘못된 요청입니다", 0)
	}

	name, ok := validateRoomName(req.Name)
	if !ok {
		return makeResponse(id, 0, "방 제목을 확인해주세요", 0)
	}

	if !validateRoomMax(int(req.Max)) {
		return makeResponse(id, 0, "최대 인원을 확인해주세요", 0)
	}

	password, err := hashPassword(req.Password)
	if err != nil {
		return makeResponse(id, 0, "방을 만들 수 없습니다", 0)
	}

	var out outbox
	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		out.send()
	}()

	item := c.Userlist.Find(id)
	if item == nil {
		return makeResponse(id, 0, "로그인이 필요합니다", 0)
	}

	roomId := c.Roomlist.Add(name, int(req.Max), req.Secret, password, item.Loginid)
	c.enterRoom(&out, item, c.Roomlist.Find(roomId))

	return makeResponse(id, 1, "", roomId)
}

// EditRoom changes the title / max / password of a room. Only the room owner
// may do it; the wait room cannot be edited. The password is stored as a
// bcrypt hash.
func (c *Chat) EditRoom(so socketio.Conn, msg string) string {
	id := so.ID()

	var req EditRoomRequest
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		return makeResponse(id, 0, "잘못된 요청입니다", 0)
	}

	var name string
	if req.Name != nil {
		var ok bool
		if name, ok = validateRoomName(*req.Name); !ok {
			return makeResponse(id, 0, "방 제목을 확인해주세요", req.Room)
		}
	}

	if req.Max != nil && !validateRoomMax(int(*req.Max)) {
		return makeResponse(id, 0, "최대 인원을 확인해주세요", req.Room)
	}

	var password string
	if req.Password != nil {
		var err error
		if password, err = hashPassword(*req.Password); err != nil {
			return makeResponse(id, 0, "방을 수정할 수 없습니다", req.Room)
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	item := c.Userlist.Find(id)
	if item == nil {
		return makeResponse(id, 0, "로그인이 필요합니다", req.Room)
	}

	room := c.Roomlist.Find(req.Room)
	if room == nil || room.Id == WAIT_ROOM || !room.IsAdmin(item) {
		log.Println("editroom refused: not room owner", id)
		return makeResponse(id, 0, "방장만 수정할 수 있습니다", req.Room)
	}

	if req.Name != nil {
		room.Name = name
	}
	if req.Max != nil {
		room.Max = int(*req.Max)
	}
	if req.Password != nil {
		room.Password = password
	}

	return makeResponse(id, 1, "", room.Id)
}

// removeFromRoom handles ban/kick: only the owner of the room may remove
// another user, and the removed user is sent back to the wait room.
// req.Id is the target user's socket id (the "id" field of chat messages).
func (c *Chat) removeFromRoom(so socketio.Conn, msg string, ban bool) string {
	id := so.ID()

	var req Request
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		return ""
	}

	var out outbox
	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		out.send()
	}()

	item := c.Userlist.Find(id)
	if item == nil {
		return ""
	}

	room := c.Roomlist.Find(req.Room)
	if room == nil || room.Id == WAIT_ROOM || !room.IsAdmin(item) {
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

	event := "kick"
	if ban {
		event = "ban"
	}
	out = append(out, delivery{target.Socket, event, c.MakeMessage(target.Id, target.Nickname, "", nil)})

	wait := c.Roomlist.Find(WAIT_ROOM)
	if wait != nil {
		c.enterRoom(&out, target, wait)
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

	var req MessageRequest
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		return ""
	}

	var out outbox
	c.mu.Lock()

	item := c.Userlist.Find(id)
	if item != nil {
		c.Broadcating(&out, item.Room, "chat", c.MakeMessage(id, item.Nickname, req.Text, &req.Option))
	}

	c.mu.Unlock()
	out.send()

	return ""
}

func (c *Chat) ChatTo(so socketio.Conn, msg string) string {
	id := so.ID()

	var req MessageToRequest
	if err := json.Unmarshal([]byte(msg), &req); err != nil {
		return ""
	}

	var out outbox
	c.mu.Lock()

	item := c.Userlist.Find(id)
	if item != nil {
		// Whispers only reach users in the sender's current room.
		room := c.Roomlist.Find(item.Room)
		if room != nil {
			if to := room.Users[req.To]; to != nil {
				out = append(out, delivery{to.Socket, "chatto", c.MakeMessage(id, item.Nickname, req.Text, &req.Option)})
			}
		}
	}

	c.mu.Unlock()
	out.send()

	return ""
}
