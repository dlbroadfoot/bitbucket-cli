package shared

import (
	"strings"

	"github.com/dlbroadfoot/bitbucket-cli/internal/gh"
)

func AuthTokenWriteable(authCfg gh.AuthConfig, hostname string) (string, bool) {
	token, src := authCfg.ActiveToken(hostname)
	return src, (token == "" || !strings.HasSuffix(src, "_TOKEN"))
}
