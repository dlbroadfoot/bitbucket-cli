package authflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetCurrentLogin(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid token",
			token: "user:apppassword",
			want:  "user",
		},
		{
			name:  "email-based token",
			token: "user@example.com:apitoken123",
			want:  "user@example.com",
		},
		{
			name:    "invalid token",
			token:   "nodelimiter",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetCurrentLogin(tt.token)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
