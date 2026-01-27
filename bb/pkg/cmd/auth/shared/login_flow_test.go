package shared

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type tinyConfig map[string]string

func (c tinyConfig) Login(host, username, token, gitProtocol string, encrypt bool) (bool, error) {
	c[fmt.Sprintf("%s:%s", host, "user")] = username
	c[fmt.Sprintf("%s:%s", host, "oauth_token")] = token
	c[fmt.Sprintf("%s:%s", host, "git_protocol")] = gitProtocol
	return false, nil
}

func (c tinyConfig) UsersForHost(hostname string) []string {
	return nil
}

func Test_GetCurrentLogin(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantUser  string
		wantError bool
	}{
		{
			name:     "valid token",
			token:    "testuser:app-password-123",
			wantUser: "testuser",
		},
		{
			name:      "invalid token format",
			token:     "no-colon-here",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := GetCurrentLogin(nil, "", tt.token)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantUser, user)
			}
		})
	}
}
