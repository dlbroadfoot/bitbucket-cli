//go:build updateable

package ghcmd

// `updateable` is a build tag that controls whether users are notified of newer
// Bitbucket CLI releases.
//
// Development builds do not generate update messages by default.
var updaterEnabled = "dlbroadfoot/bitbucket-cli"
