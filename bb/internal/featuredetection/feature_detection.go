// Package featuredetection provides feature detection for Bitbucket.
// Unlike GitHub, Bitbucket doesn't have GraphQL and we rely on REST API versioning.
// This package provides stubs for compatibility with code migrated from gh CLI.
package featuredetection

import (
	"github.com/cli/bb/v2/internal/bbrepo"
)

// Detector detects features available on a Bitbucket instance.
type Detector interface {
	IssueFeatures() (IssueFeatures, error)
	PullRequestFeatures() (PullRequestFeatures, error)
	RepositoryFeatures() (RepositoryFeatures, error)
}

// IssueFeatures represents features available for issues.
type IssueFeatures struct {
	// Bitbucket Cloud always supports issue tracker (if enabled)
	StateReason bool
}

// PullRequestFeatures represents features available for pull requests.
type PullRequestFeatures struct {
	// Bitbucket Cloud supports these features
	MergeQueue    bool
	CheckRunEvent bool
}

// RepositoryFeatures represents features available for repositories.
type RepositoryFeatures struct {
	// Bitbucket Cloud supports these features
	AutoMerge            bool
	VisibilityField      bool
	IssueTemplateMutation bool
	IssueTemplateQuery   bool
}

// NewDetector creates a new feature detector for Bitbucket.
func NewDetector(_ interface{}, _ bbrepo.Interface) Detector {
	return &detector{}
}

type detector struct{}

func (d *detector) IssueFeatures() (IssueFeatures, error) {
	// Bitbucket Cloud features
	return IssueFeatures{
		StateReason: false, // Bitbucket doesn't have state reason
	}, nil
}

func (d *detector) PullRequestFeatures() (PullRequestFeatures, error) {
	// Bitbucket Cloud features
	return PullRequestFeatures{
		MergeQueue:    false, // Bitbucket doesn't have merge queue (yet)
		CheckRunEvent: false, // Different CI system
	}, nil
}

func (d *detector) RepositoryFeatures() (RepositoryFeatures, error) {
	// Bitbucket Cloud features
	return RepositoryFeatures{
		AutoMerge:            true,  // Bitbucket supports auto-merge
		VisibilityField:      true,  // Bitbucket has visibility settings
		IssueTemplateMutation: false,
		IssueTemplateQuery:   false,
	}, nil
}
