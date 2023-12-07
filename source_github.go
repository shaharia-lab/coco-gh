package cocogh

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/shurcooL/githubv4"
)

// Paths is a struct to keep track of the status of file paths
type Paths struct {
	Added    []string
	Removed  []string
	Modified []string
}

// GitHubFilter is a filter used for GitHub file search.
type GitHubFilter struct {
	FilePath  string
	FileTypes []string
}

// GitHubConfig holds the configuration for the GitHub client such as the repositories and branches to work with.
type GitHubConfig struct {
	Owner         string
	Repositories  []string
	DefaultBranch string
	Filter        GitHubFilter
}

// GitHubClient stores a repo's GitHub client and its related configurations.
type GitHubClient struct {
	Client        GHClient
	Configuration GitHubConfig
}

// GHClient is an interface to help test the GitHub Client.
type GHClient interface {
	Query(ctx context.Context, q interface{}, variables map[string]interface{}) error
	ListCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error)
	GetCommit(ctx context.Context, owner, repo, sha string, opts *github.ListOptions) (*github.RepositoryCommit, *github.Response, error)
}

// GHQueryForListFiles holds the structure of the query to fetch all files from a repository.
type GHQueryForListFiles struct {
	Repository struct {
		Object struct {
			Tree struct {
				Entries []struct {
					Name string
					Path string
					Type string
				}
			} `graphql:"... on Tree"`
		} `graphql:"object(expression: $expression)"`
	} `graphql:"repository(owner: $owner, name: $name)"`
}

// NewGitHubClient creates a new GitHubClient with the given GHClient and configuration.
func NewGitHubClient(ghClient GHClient, configuration GitHubConfig) *GitHubClient {
	return &GitHubClient{
		Client:        ghClient,
		Configuration: configuration,
	}
}

// GetFilePathsForRepositories fetches the file paths from all the repositories specified in the GitHubClient.
func (c *GitHubClient) GetFilePathsForRepositories() ([]string, error) {
	var files []string
	for _, repo := range c.Configuration.Repositories {
		fs, err := c.getFilePathsForRepo(c.Configuration.Owner, repo, fmt.Sprintf("%s:%s", c.Configuration.DefaultBranch, c.Configuration.Filter.FilePath))
		if err != nil {
			return nil, err
		}
		files = append(files, fs...)
	}

	if len(c.Configuration.Filter.FileTypes) == 0 {
		return files, nil
	}

	var filteredFiles []string
	for i, file := range files {
		if !c.hasFileType(file, c.Configuration.Filter.FileTypes) {
			continue
		}
		filteredFiles = append(filteredFiles, files[i])
	}

	return filteredFiles, nil
}

// GetChangedFilePathsSince fetches the file paths from all repositories that have been changed in the specified duration (in hours).
func (c *GitHubClient) GetChangedFilePathsSince(hoursSince int) (Paths, error) {
	ctx := context.Background()

	now := time.Now()
	dayToHour := 24 * hoursSince
	specifiedTime := now.Add(time.Hour * time.Duration(-dayToHour))

	opt := &github.CommitsListOptions{
		Since: specifiedTime,
		Path:  c.Configuration.Filter.FilePath,
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var paths Paths

	for _, repo := range c.Configuration.Repositories {
		commitPaths, err := c.getChangedFilePathsForRepo(ctx, repo, opt)
		if err != nil {
			return Paths{}, err
		}
		paths.Added = append(paths.Added, commitPaths.Added...)
		paths.Removed = append(paths.Removed, commitPaths.Removed...)
		paths.Modified = append(paths.Modified, commitPaths.Modified...)
	}

	return paths, nil
}

// getFilePathsForRepo fetches the file paths in a GitHub repository.
func (c *GitHubClient) getFilePathsForRepo(owner, name, expression string) ([]string, error) {
	var query GHQueryForListFiles
	variables := map[string]interface{}{
		"owner":      githubv4.String(owner),
		"name":       githubv4.String(name),
		"expression": githubv4.String(expression),
	}

	err := c.Client.Query(context.Background(), &query, variables)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range query.Repository.Object.Tree.Entries {
		if entry.Type == "blob" {
			files = append(files, entry.Path)
		} else if entry.Type == "tree" {
			subFiles, err := c.getFilePathsForRepo(owner, name, expression+"/"+entry.Name)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		}
	}

	return files, nil
}

// hasFileType determines if a filename ends with certain file types.
func (c *GitHubClient) hasFileType(fileName string, fileTypes []string) bool {
	for _, fileType := range fileTypes {
		if strings.HasSuffix(fileName, fileType) {
			return true
		}
	}
	return false
}

// getChangedFilePathsForRepo fetches the file paths in a repository that have been changed.
func (c *GitHubClient) getChangedFilePathsForRepo(ctx context.Context, repo string, opt *github.CommitsListOptions) (Paths, error) {
	var paths Paths

	commits, _, err := c.Client.ListCommits(ctx, c.Configuration.Owner, repo, opt)
	if err != nil {
		return paths, err
	}

	directory := c.Configuration.Filter.FilePath

	for _, commit := range commits {
		commitDetails, _, err := c.Client.GetCommit(ctx, c.Configuration.Owner, repo, *commit.SHA, nil)
		if err != nil {
			return paths, err
		}

		for _, file := range commitDetails.Files {
			if strings.HasPrefix(*file.Filename, directory) {
				switch file.GetStatus() {
				case "removed":
					paths.Removed = append(paths.Removed, *file.Filename)
				case "added":
					paths.Added = append(paths.Added, *file.Filename)
				case "modified", "changed":
					paths.Modified = append(paths.Modified, *file.Filename)
				case "renamed":
					paths.Removed = append(paths.Removed, *file.PreviousFilename)
					paths.Added = append(paths.Modified, *file.Filename)
				case "copied":
					paths.Added = append(paths.Modified, *file.Filename)
				}
			}
		}
	}

	return paths, nil
}
