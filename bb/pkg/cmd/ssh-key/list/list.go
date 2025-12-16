package list

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/tableprinter"
	"github.com/dlbroadfoot/bitbucket-cli/internal/text"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List SSH keys in your Bitbucket account",
		Long: heredoc.Doc(`
			List SSH keys registered in your Bitbucket account.

			Shows all SSH keys that can be used for authentication.
		`),
		Example: heredoc.Doc(`
			$ bb ssh-key list
		`),
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}

	return cmd
}

// SSHKey represents an SSH key
type SSHKey struct {
	UUID      string `json:"uuid"`
	Key       string `json:"key"`
	Label     string `json:"label"`
	Comment   string `json:"comment"`
	CreatedOn string `json:"created_on"`
	Links     struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

// SSHKeyList represents a paginated list of SSH keys
type SSHKeyList struct {
	Size     int      `json:"size"`
	Page     int      `json:"page"`
	PageLen  int      `json:"pagelen"`
	Next     string   `json:"next"`
	Previous string   `json:"previous"`
	Values   []SSHKey `json:"values"`
}

func listRun(opts *ListOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	opts.IO.StartProgressIndicator()
	keys, err := fetchSSHKeys(httpClient)
	opts.IO.StopProgressIndicator()

	if err != nil {
		return err
	}

	if len(keys) == 0 {
		fmt.Fprintln(opts.IO.Out, "No SSH keys found")
		return nil
	}

	return printKeys(opts.IO, keys)
}

func fetchSSHKeys(client *http.Client) ([]SSHKey, error) {
	apiClient := api.NewClientFromHTTP(client)

	path := "user/ssh-keys?pagelen=100"

	var result SSHKeyList
	err := apiClient.Get("bitbucket.org", path, &result)
	if err != nil {
		return nil, err
	}

	return result.Values, nil
}

func printKeys(io *iostreams.IOStreams, keys []SSHKey) error {
	tp := tableprinter.New(io, tableprinter.WithHeader("ID", "TITLE", "KEY", "ADDED"))

	for _, k := range keys {
		// Extract the key ID (UUID without braces)
		id := k.UUID
		if len(id) > 8 {
			id = id[:8] + "..."
		}
		tp.AddField(id)

		// Title/Label
		label := k.Label
		if label == "" {
			label = k.Comment
		}
		if label == "" {
			label = "-"
		}
		tp.AddField(label)

		// Truncated key (just show the type and first few chars)
		keyPreview := truncateKey(k.Key)
		tp.AddField(keyPreview)

		// Created date
		if t, err := time.Parse(time.RFC3339, k.CreatedOn); err == nil {
			tp.AddField(text.FuzzyAgo(time.Now(), t))
		} else {
			tp.AddField("-")
		}

		tp.EndRow()
	}

	return tp.Render()
}

func truncateKey(key string) string {
	if len(key) > 30 {
		return key[:27] + "..."
	}
	return key
}
