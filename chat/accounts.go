package chat

import (
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// Accounts keeps loginid -> bcrypt password hash.
//
// There is no persistent user store in this server, so the first successful
// login for a loginid registers it (the password is hashed with bcrypt) and
// every later login for that loginid must present the same password.
// Accounts live in memory and are lost when the server restarts.
type Accounts struct {
	mu    sync.Mutex
	items map[string][]byte
}

func NewAccounts() *Accounts {
	return &Accounts{items: make(map[string][]byte)}
}

// Verify returns true when the password matches the stored hash, or when the
// loginid is new and was registered with this password.
func (c *Accounts) Verify(loginid string, password string) bool {
	if loginid == "" || password == "" {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if hash, ok := c.items[loginid]; ok {
		return bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false
	}
	c.items[loginid] = hash

	return true
}

// hashPassword hashes a room password. An empty password means "no password".
func hashPassword(password string) (string, error) {
	if password == "" {
		return "", nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func checkPassword(hash string, password string) bool {
	if hash == "" {
		return false
	}

	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
