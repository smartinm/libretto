// Copyright 2015 Apcera Inc. All rights reserved.

package ssh

import (
	"testing"

	cssh "golang.org/x/crypto/ssh"
)

func requireMockedClient() SSHClient {
	c := SSHClient{}
	c.Creds = &Credentials{}
	dial = func(p string, a string, c *cssh.ClientConfig) (*cssh.Client, error) {
		return nil, nil
	}
	readPrivateKey = func(path string) (cssh.AuthMethod, error) {
		return nil, nil
	}
	return c
}

// TestConnectNoUsername tests that an error is returned if no username is provided.
func TestConnectNoUsername(t *testing.T) {
	c := requireMockedClient()
	err := c.Connect()
	if err != ErrInvalidUsername {
		t.Logf("Invalid error type returned %s", err)
		t.Fail()
	}
}

// TestConnectNoPassword tests that an error is returned if no password or key is provided.
func TestConnectNoPassword(t *testing.T) {
	c := requireMockedClient()
	c.Creds.SSHUser = "foo"
	err := c.Connect()
	if err != ErrInvalidAuth {
		t.Logf("Invalid error type returned %s", err)
		t.Fail()
	}
}

// TestConnectAuthPrecedence tests that key based auth takes precedence over password based auth
func TestConnectAuthPrecedence(t *testing.T) {
	c := requireMockedClient()
	count := 0

	c.Creds = &Credentials{
		SSHUser:       "test",
		SSHPassword:   "test",
		SSHPrivateKey: "/foo",
	}

	readPrivateKey = func(path string) (cssh.AuthMethod, error) {
		count++
		return nil, nil
	}
	err := c.Connect()
	if err != nil {
		t.Logf("Expected nil error, got %s", err)
		t.Fail()
	}
	if count != 1 {
		t.Logf("Should have called the password key method %d times", count)
		t.Fail()
	}
}
