package shared

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/dlbroadfoot/bitbucket-cli/internal/bbrepo"
)

// Pipeline represents a Bitbucket pipeline
type Pipeline struct {
	UUID        string  `json:"uuid"`
	BuildNumber int     `json:"build_number"`
	Creator     *User   `json:"creator,omitempty"`
	Repository  *Repo   `json:"repository,omitempty"`
	Target      *Target `json:"target,omitempty"`
	Trigger     *struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"trigger,omitempty"`
	State       *State  `json:"state,omitempty"`
	CreatedOn   string  `json:"created_on"`
	CompletedOn string  `json:"completed_on,omitempty"`
	DurationIn  int     `json:"duration_in_seconds,omitempty"`
	Links       Links   `json:"links"`
}

type User struct {
	DisplayName string `json:"display_name"`
	UUID        string `json:"uuid"`
	AccountID   string `json:"account_id"`
	Nickname    string `json:"nickname"`
	Links       Links  `json:"links"`
}

type Repo struct {
	FullName string `json:"full_name"`
	UUID     string `json:"uuid"`
	Name     string `json:"name"`
	Links    Links  `json:"links"`
}

type Target struct {
	Type     string  `json:"type"`
	RefType  string  `json:"ref_type,omitempty"`
	RefName  string  `json:"ref_name,omitempty"`
	Selector *struct {
		Type    string `json:"type"`
		Pattern string `json:"pattern"`
	} `json:"selector,omitempty"`
	Commit *struct {
		Hash    string `json:"hash"`
		Message string `json:"message,omitempty"`
	} `json:"commit,omitempty"`
	PullRequest *struct {
		ID    int    `json:"id"`
		Title string `json:"title,omitempty"`
	} `json:"pullrequest,omitempty"`
}

type State struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Result *struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"result,omitempty"`
	Stage *struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"stage,omitempty"`
}

type Links struct {
	HTML struct {
		Href string `json:"href"`
	} `json:"html"`
	Self struct {
		Href string `json:"href"`
	} `json:"self"`
	Steps struct {
		Href string `json:"href"`
	} `json:"steps,omitempty"`
}

// PipelineList represents a paginated list of pipelines
type PipelineList struct {
	Size     int        `json:"size"`
	Page     int        `json:"page"`
	PageLen  int        `json:"pagelen"`
	Next     string     `json:"next"`
	Previous string     `json:"previous"`
	Values   []Pipeline `json:"values"`
}

// Step represents a pipeline step
type Step struct {
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
	State      *State `json:"state,omitempty"`
	StartedOn  string `json:"started_on,omitempty"`
	CompletedOn string `json:"completed_on,omitempty"`
	DurationIn int    `json:"duration_in_seconds,omitempty"`
	Image      *struct {
		Name string `json:"name"`
	} `json:"image,omitempty"`
	ScriptCommands []struct {
		Name    string `json:"name"`
		Command string `json:"command"`
	} `json:"script_commands,omitempty"`
	Links Links `json:"links"`
}

// StepList represents a paginated list of steps
type StepList struct {
	Size     int    `json:"size"`
	Page     int    `json:"page"`
	PageLen  int    `json:"pagelen"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Values   []Step `json:"values"`
}

// StepLog represents logs for a step
type StepLog struct {
	Name string `json:"name"`
	Log  string `json:"log"`
}

// HTMLURL returns the web URL for the pipeline
func (p *Pipeline) HTMLURL() string {
	return p.Links.HTML.Href
}

// StatusString returns a human-readable status
func (p *Pipeline) StatusString() string {
	if p.State == nil {
		return "Unknown"
	}

	switch p.State.Name {
	case "PENDING":
		return "Pending"
	case "IN_PROGRESS":
		if p.State.Stage != nil {
			return fmt.Sprintf("Running (%s)", p.State.Stage.Name)
		}
		return "Running"
	case "COMPLETED":
		if p.State.Result != nil {
			switch p.State.Result.Name {
			case "SUCCESSFUL":
				return "Successful"
			case "FAILED":
				return "Failed"
			case "ERROR":
				return "Error"
			case "STOPPED":
				return "Stopped"
			case "EXPIRED":
				return "Expired"
			}
		}
		return "Completed"
	case "HALTED":
		return "Halted"
	case "PAUSED":
		return "Paused"
	default:
		return p.State.Name
	}
}

// IsRunning returns true if the pipeline is currently running
func (p *Pipeline) IsRunning() bool {
	if p.State == nil {
		return false
	}
	return p.State.Name == "IN_PROGRESS" || p.State.Name == "PENDING" || p.State.Name == "PAUSED"
}

// RefName returns the branch or tag name
func (p *Pipeline) RefName() string {
	if p.Target == nil {
		return ""
	}
	return p.Target.RefName
}

// CommitHash returns the short commit hash
func (p *Pipeline) CommitHash() string {
	if p.Target == nil || p.Target.Commit == nil {
		return ""
	}
	hash := p.Target.Commit.Hash
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}

// ParsePipelineArg parses a pipeline argument which can be a build number or URL
func ParsePipelineArg(arg string) (int, bbrepo.Interface, error) {
	// Try parsing as a number first
	if num, err := strconv.Atoi(arg); err == nil {
		return num, nil, nil
	}

	// Try parsing as a URL
	// Format: https://bitbucket.org/WORKSPACE/REPO/addon/pipelines/home#!/results/123
	re := regexp.MustCompile(`^https?://([^/]+)/([^/]+)/([^/]+)/addon/pipelines/home#!/results/(\d+)`)
	matches := re.FindStringSubmatch(arg)
	if matches != nil {
		host := matches[1]
		workspace := matches[2]
		repoSlug := matches[3]
		num, _ := strconv.Atoi(matches[4])

		repo := bbrepo.NewWithHost(workspace, repoSlug, host)
		return num, repo, nil
	}

	return 0, nil, fmt.Errorf("invalid pipeline argument: %s", arg)
}
