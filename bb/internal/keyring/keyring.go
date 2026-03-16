// Package keyring is a simple wrapper that adds timeouts to the zalando/go-keyring package.
package keyring

import (
	"errors"
	"fmt"
	"time"

	"github.com/zalando/go-keyring"
)

var ErrNotFound = errors.New("secret not found in keyring")

type TimeoutError struct {
	message string
}

func (e *TimeoutError) Error() string {
	return e.message
}

// Set secret in keyring for user.
func Set(service, user, secret string) error {
	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		ch <- keyring.Set(service, user, secret)
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(Timeout()):
		return &TimeoutError{timeoutMessage("set")}
	}
}

// Get secret from keyring given service and user name.
func Get(service, user string) (string, error) {
	ch := make(chan struct {
		val string
		err error
	}, 1)
	go func() {
		defer close(ch)
		val, err := keyring.Get(service, user)
		ch <- struct {
			val string
			err error
		}{val, err}
	}()
	select {
	case res := <-ch:
		if errors.Is(res.err, keyring.ErrNotFound) {
			return "", ErrNotFound
		}
		return res.val, res.err
	case <-time.After(Timeout()):
		return "", &TimeoutError{timeoutMessage("get")}
	}
}

// Delete secret from keyring.
func Delete(service, user string) error {
	ch := make(chan error, 1)
	go func() {
		defer close(ch)
		ch <- keyring.Delete(service, user)
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(Timeout()):
		return &TimeoutError{timeoutMessage("delete")}
	}
}

func timeoutMessage(op string) string {
	msg := fmt.Sprintf("timeout while trying to %s secret in keyring", op)
	if IsHeadless() {
		msg += "\nhint: use --insecure-storage or set BB_TOKEN to avoid keyring in headless environments"
	} else {
		msg += fmt.Sprintf("\nhint: increase timeout with BB_KEYRING_TIMEOUT (current: %s)", Timeout())
	}
	return msg
}

func MockInit() {
	keyring.MockInit()
}

func MockInitWithError(err error) {
	keyring.MockInitWithError(err)
}
