package add

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type AddOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams

	Title   string
	KeyFile string
}

func NewCmdAdd(f *cmdutil.Factory, runF func(*AddOptions) error) *cobra.Command {
	opts := &AddOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
	}

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add an SSH key to your Bitbucket account",
		Long: heredoc.Doc(`
			Add an SSH key to your Bitbucket account.

			The key can be read from:
			- A file specified with --key-file
			- Standard input (if key-file is -)
			- Default location (~/.ssh/id_rsa.pub, ~/.ssh/id_ed25519.pub)
		`),
		Example: heredoc.Doc(`
			$ bb ssh-key add --title "My Laptop"
			$ bb ssh-key add --title "Work Computer" --key-file ~/.ssh/work_key.pub
			$ cat ~/.ssh/id_rsa.pub | bb ssh-key add --title "My Key" --key-file -
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Title == "" {
				hostname, err := os.Hostname()
				if err == nil {
					opts.Title = hostname
				} else {
					opts.Title = "SSH Key"
				}
			}

			if runF != nil {
				return runF(opts)
			}
			return addRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Title for the SSH key (defaults to hostname)")
	cmd.Flags().StringVarP(&opts.KeyFile, "key-file", "k", "", "Path to public key file (- for stdin)")

	return cmd
}

func addRun(opts *AddOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	// Read the SSH key
	keyContent, err := readKey(opts)
	if err != nil {
		return err
	}

	keyContent = strings.TrimSpace(keyContent)
	if keyContent == "" {
		return fmt.Errorf("SSH key is empty")
	}

	opts.IO.StartProgressIndicator()
	key, err := addSSHKey(httpClient, opts.Title, keyContent)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if opts.IO.IsStdoutTTY() {
		cs := opts.IO.ColorScheme()
		fmt.Fprintf(opts.IO.Out, "%s Added SSH key %s\n", cs.SuccessIcon(), cs.Bold(opts.Title))
		if key.UUID != "" {
			fmt.Fprintf(opts.IO.Out, "Key ID: %s\n", key.UUID)
		}
	}

	return nil
}

func readKey(opts *AddOptions) (string, error) {
	// Read from stdin
	if opts.KeyFile == "-" {
		data, err := io.ReadAll(opts.IO.In)
		if err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		return string(data), nil
	}

	// Read from specified file
	if opts.KeyFile != "" {
		data, err := os.ReadFile(opts.KeyFile)
		if err != nil {
			return "", fmt.Errorf("failed to read key file: %w", err)
		}
		return string(data), nil
	}

	// Try default locations
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	defaultKeys := []string{
		filepath.Join(home, ".ssh", "id_ed25519.pub"),
		filepath.Join(home, ".ssh", "id_rsa.pub"),
		filepath.Join(home, ".ssh", "id_ecdsa.pub"),
	}

	for _, keyPath := range defaultKeys {
		if _, err := os.Stat(keyPath); err == nil {
			data, err := os.ReadFile(keyPath)
			if err != nil {
				continue
			}
			if opts.IO.IsStdoutTTY() {
				fmt.Fprintf(opts.IO.ErrOut, "Using key from %s\n", keyPath)
			}
			return string(data), nil
		}
	}

	return "", fmt.Errorf("no SSH key found; specify one with --key-file")
}

type sshKeyPayload struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type sshKeyResponse struct {
	UUID  string `json:"uuid"`
	Label string `json:"label"`
	Key   string `json:"key"`
}

func addSSHKey(client *http.Client, title, key string) (*sshKeyResponse, error) {
	apiClient := api.NewClientFromHTTP(client)

	path := "user/ssh-keys"

	payload := sshKeyPayload{
		Key:   key,
		Label: title,
	}

	var result sshKeyResponse
	err := apiClient.Post("bitbucket.org", path, payload, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
