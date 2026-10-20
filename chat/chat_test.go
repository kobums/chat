package chat

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	socketio "github.com/googollee/go-socket.io"
)

type fakeConn struct {
	socketio.Conn
	id string

	mu     sync.Mutex
	events []string
}

func (f *fakeConn) ID() string { return f.id }

func (f *fakeConn) Emit(event string, v ...interface{}) {
	f.mu.Lock()
	f.events = append(f.events, event)
	f.mu.Unlock()
}

func login(t *testing.T, c *Chat, so *fakeConn, loginid string) {
	t.Helper()
	msg, _ := json.Marshal(LoginRequest{Loginid: loginid, Password: "pw-" + loginid, Nickname: loginid})
	var res Response
	if err := json.Unmarshal([]byte(c.Login(so, string(msg))), &res); err != nil || res.Status != 1 {
		t.Fatalf("login %s failed: %v %+v", loginid, err, res)
	}
}

func roomOf(c *Chat, so *fakeConn) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Userlist.Find(so.id).Room
}

func roomsContaining(c *Chat, so *fakeConn) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, r := range c.Roomlist.Items {
		if r.Users[so.id] != nil {
			n++
		}
	}
	return n
}

func TestMakeJoinExitRoom(t *testing.T) {
	c := NewChat()
	owner := &fakeConn{id: "1"}
	guest := &fakeConn{id: "2"}
	login(t, c, owner, "owner")
	login(t, c, guest, "guest")

	// max as a string, as the web client sends it.
	var res Response
	json.Unmarshal([]byte(c.MakeRoom(owner, `{"name":"room","max":"2","password":"secret"}`)), &res)
	if res.Status != 1 || res.Room == WAIT_ROOM {
		t.Fatalf("makeroom failed: %+v", res)
	}
	roomId := res.Room
	if roomOf(c, owner) != roomId || roomsContaining(c, owner) != 1 {
		t.Fatalf("owner should be only in the new room")
	}

	c.JoinRoom(guest, fmt.Sprintf(`{"room":%d,"password":"wrong"}`, roomId))
	if roomOf(c, guest) != WAIT_ROOM {
		t.Fatalf("wrong password must not join")
	}

	c.JoinRoom(guest, fmt.Sprintf(`{"room":%d,"password":"secret"}`, roomId))
	if roomOf(c, guest) != roomId || roomsContaining(c, guest) != 1 {
		t.Fatalf("guest should be only in the room")
	}

	// Chat goes to the current room.
	c.Chat(guest, `{"text":"hi"}`)

	c.ExitRoom(guest, fmt.Sprintf(`{"room":%d}`, roomId))
	if roomOf(c, guest) != WAIT_ROOM || roomsContaining(c, guest) != 1 {
		t.Fatalf("guest should be back in the wait room")
	}

	// Non-owner edit is refused.
	json.Unmarshal([]byte(c.EditRoom(guest, fmt.Sprintf(`{"room":%d,"name":"hacked"}`, roomId))), &res)
	if res.Status != 0 || c.Roomlist.Find(roomId).Name != "room" {
		t.Fatalf("non-owner edit must be refused")
	}

	json.Unmarshal([]byte(c.EditRoom(owner, fmt.Sprintf(`{"room":%d,"name":"renamed","max":5,"password":""}`, roomId))), &res)
	r := c.Roomlist.Find(roomId)
	if res.Status != 1 || r.Name != "renamed" || r.Max != 5 || r.HasPassword() {
		t.Fatalf("owner edit failed: %+v %+v", res, r)
	}

	// Last user leaving removes the room.
	c.ExitRoom(owner, fmt.Sprintf(`{"room":%d}`, roomId))
	if c.Roomlist.Find(roomId) != nil {
		t.Fatalf("empty room should be removed")
	}
}

func TestConcurrentHandlers(t *testing.T) {
	c := NewChat()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			so := &fakeConn{id: fmt.Sprint(i)}
			loginid := fmt.Sprint("u", i)
			msg, _ := json.Marshal(LoginRequest{Loginid: loginid, Password: "pw", Nickname: loginid})
			c.Login(so, string(msg))
			c.MakeRoom(so, `{"name":"r","max":10}`)
			c.ListRoom(so, "")
			c.JoinRoom(so, `{"room":1}`)
			c.Chat(so, `{"text":"x"}`)
			c.ExitRoom(so, `{"room":1}`)
			c.Disconnect(so)
		}(i)
	}
	wg.Wait()
}
