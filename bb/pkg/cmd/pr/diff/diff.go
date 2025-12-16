package diff

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/MakeNowJust/heredoc"
	"github.com/dlbroadfoot/bitbucket-cli/api"
	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
	"github.com/dlbroadfoot/bitbucket-cli/internal/browser"
	"github.com/dlbroadfoot/bitbucket-cli/internal/text"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmd/pr/shared"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/cmdutil"
	"github.com/dlbroadfoot/bitbucket-cli/pkg/iostreams"
	"github.com/spf13/cobra"
	"golang.org/x/text/transform"
)

type DiffOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (bbrepo.Interface, error)
	Browser    browser.Browser

	SelectorArg string
	UseColor    bool
	NameOnly    bool
	BrowserMode bool
}

func NewCmdDiff(f *cmdutil.Factory, runF func(*DiffOptions) error) *cobra.Command {
	opts := &DiffOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		BaseRepo:   f.BaseRepo,
		Browser:    f.Browser,
	}

	var colorFlag string

	cmd := &cobra.Command{
		Use:   "diff [<number> | <url>]",
		Short: "View changes in a pull request",
		Long: heredoc.Doc(`
			View changes in a pull request.

			Without an argument, the pull request that belongs to the current branch
			is displayed.

			With --web flag, open the pull request diff in a web browser instead.
		`),
		Example: heredoc.Doc(`
			# View diff for pull request #123
			$ bb pr diff 123

			# View diff with color output
			$ bb pr diff 123 --color always

			# View only changed file names
			$ bb pr diff 123 --name-only

			# Open diff in browser
			$ bb pr diff 123 --web
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.SelectorArg = args[0]
			}

			switch colorFlag {
			case "always":
				opts.UseColor = true
			case "auto":
				opts.UseColor = opts.IO.ColorEnabled()
			case "never":
				opts.UseColor = false
			default:
				return fmt.Errorf("unsupported color %q", colorFlag)
			}

			if runF != nil {
				return runF(opts)
			}
			return diffRun(opts)
		},
	}

	cmdutil.StringEnumFlag(cmd, &colorFlag, "color", "", "auto", []string{"always", "never", "auto"}, "Use color in diff output")
	cmd.Flags().BoolVar(&opts.NameOnly, "name-only", false, "Display only names of changed files")
	cmd.Flags().BoolVarP(&opts.BrowserMode, "web", "w", false, "Open the pull request diff in the browser")

	return cmd
}

func diffRun(opts *DiffOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}

	// Parse the PR argument first to check if it contains repo info
	prID, prRepo, err := shared.ParsePRArg(opts.SelectorArg)
	if err != nil {
		return err
	}

	// Use the repo from URL if provided, otherwise resolve from git remotes
	var repo bbrepo.Interface
	if prRepo != nil {
		repo = prRepo
	} else {
		repo, err = opts.BaseRepo()
		if err != nil {
			return err
		}
	}

	if opts.BrowserMode {
		openURL := fmt.Sprintf("https://%s/%s/%s/pull-requests/%d/diff",
			repo.RepoHost(), repo.RepoWorkspace(), repo.RepoSlug(), prID)
		if opts.IO.IsStdoutTTY() {
			fmt.Fprintf(opts.IO.ErrOut, "Opening %s in your browser.\n", text.DisplayURL(openURL))
		}
		return opts.Browser.Browse(openURL)
	}

	diffReadCloser, err := fetchDiff(httpClient, repo, prID)
	if err != nil {
		return fmt.Errorf("could not find pull request diff: %w", err)
	}
	defer diffReadCloser.Close()

	var diff io.Reader = diffReadCloser
	if opts.IO.IsStdoutTTY() {
		diff = sanitizedReader(diff)
	}

	if err := opts.IO.StartPager(); err == nil {
		defer opts.IO.StopPager()
	} else {
		fmt.Fprintf(opts.IO.ErrOut, "failed to start pager: %v\n", err)
	}

	if opts.NameOnly {
		return changedFilesNames(opts.IO.Out, diff)
	}

	if !opts.UseColor {
		_, err = io.Copy(opts.IO.Out, diff)
		return err
	}

	return colorDiffLines(opts.IO.Out, diff)
}

func fetchDiff(httpClient *http.Client, repo bbrepo.Interface, prID int) (io.ReadCloser, error) {
	// Bitbucket diff endpoint
	url := fmt.Sprintf("%srepositories/%s/%s/pullrequests/%d/diff",
		api.RESTPrefix(repo.RepoHost()),
		repo.RepoWorkspace(),
		repo.RepoSlug(),
		prID,
	)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/plain")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("pull request #%d not found", prID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, api.HandleHTTPError(resp)
	}

	return resp.Body, nil
}

const lineBufferSize = 4096

var (
	colorHeader   = []byte("\x1b[1;38m")
	colorAddition = []byte("\x1b[32m")
	colorRemoval  = []byte("\x1b[31m")
	colorReset    = []byte("\x1b[m")
)

func colorDiffLines(w io.Writer, r io.Reader) error {
	diffLines := bufio.NewReaderSize(r, lineBufferSize)
	wasPrefix := false
	needsReset := false

	for {
		diffLine, isPrefix, err := diffLines.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("error reading pull request diff: %w", err)
		}

		var color []byte
		if !wasPrefix {
			if isHeaderLine(diffLine) {
				color = colorHeader
			} else if isAdditionLine(diffLine) {
				color = colorAddition
			} else if isRemovalLine(diffLine) {
				color = colorRemoval
			}
		}

		if color != nil {
			if _, err := w.Write(color); err != nil {
				return err
			}
			needsReset = true
		}

		if _, err := w.Write(diffLine); err != nil {
			return err
		}

		if !isPrefix {
			if needsReset {
				if _, err := w.Write(colorReset); err != nil {
					return err
				}
				needsReset = false
			}
			if _, err := w.Write([]byte{'\n'}); err != nil {
				return err
			}
		}
		wasPrefix = isPrefix
	}
	return nil
}

var diffHeaderPrefixes = []string{"+++", "---", "diff", "index"}

func isHeaderLine(l []byte) bool {
	dl := string(l)
	for _, p := range diffHeaderPrefixes {
		if strings.HasPrefix(dl, p) {
			return true
		}
	}
	return false
}

func isAdditionLine(l []byte) bool {
	return len(l) > 0 && l[0] == '+'
}

func isRemovalLine(l []byte) bool {
	return len(l) > 0 && l[0] == '-'
}

func changedFilesNames(w io.Writer, r io.Reader) error {
	diff, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	pattern := regexp.MustCompile(`(?:^|\n)diff\s--git.*\s(["]?)b/(.*)`)
	matches := pattern.FindAllStringSubmatch(string(diff), -1)

	for _, val := range matches {
		name := strings.TrimSpace(val[1] + val[2])
		if _, err := w.Write([]byte(name + "\n")); err != nil {
			return err
		}
	}

	return nil
}

func sanitizedReader(r io.Reader) io.Reader {
	return transform.NewReader(r, sanitizer{})
}

// sanitizer replaces non-printable characters with their printable representations
type sanitizer struct{ transform.NopResetter }

// Transform implements transform.Transformer.
func (t sanitizer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for r, size := rune(0), 0; nSrc < len(src); {
		if r = rune(src[nSrc]); r < utf8.RuneSelf {
			size = 1
		} else if r, size = utf8.DecodeRune(src[nSrc:]); size == 1 && !atEOF && !utf8.FullRune(src[nSrc:]) {
			// Invalid rune.
			err = transform.ErrShortSrc
			break
		}

		if isPrint(r) {
			if nDst+size > len(dst) {
				err = transform.ErrShortDst
				break
			}
			for i := 0; i < size; i++ {
				dst[nDst] = src[nSrc]
				nDst++
				nSrc++
			}
			continue
		} else {
			nSrc += size
		}

		replacement := fmt.Sprintf("\\u{%02x}", r)

		if nDst+len(replacement) > len(dst) {
			err = transform.ErrShortDst
			break
		}

		for _, c := range replacement {
			dst[nDst] = byte(c)
			nDst++
		}
	}
	return
}

// isPrint reports if a rune is safe to be printed to a terminal
func isPrint(r rune) bool {
	return r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r)
}
