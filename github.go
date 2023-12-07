// Package cocogh to collect GitHub contents
package cocogh

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/shurcooL/githubv4"
)

// Paths represents a collection of file paths that have been added, removed, or modified.
type Paths struct {
	Added    []string
	Removed  []string
	Modified []string
}

// GitHubFilter represents a filter used to narrow down the file paths in a GitHub repository based on the file path and file types.
type GitHubFilter struct {
	FilePath  string
	FileTypes []string
}

// GitHubConfig represents the configuration for GitHub repositories.
//
// Owner represents the owner of the repositories.
// Repositories represents a list of repository names.
// DefaultBranch represents the default branch for the repositories.
// Filter represents the filter to apply when fetching file paths from the repositories.
type GitHubConfig struct {
	Owner         string
	Repositories  []string
	DefaultBranch string
	Filter        GitHubFilter
}

// GitHub stores CommitOpsClient, GraphQLClient and configuration.
type GitHub struct {
	Configuration GitHubConfig

	graphQLClient   GraphQLClient
	commitOpsClient CommitOpsClient
}

// GraphQLClient is an interface to help test the GitHub GraphQLClient.
type GraphQLClient interface {
	Query(ctx context.Context, q interface{}, variables map[string]interface{}) error
}

// CommitOpsClient is an interface to help test the GitHub GitHubCommitsOpsClient.
type CommitOpsClient interface {
	ListCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error)
	GetCommit(ctx context.Context, owner, repo, sha string, opts *github.ListOptions) (*github.RepositoryCommit, *github.Response, error)
}

// GitHubCommitsOpsClient is a type that represents an operations client for GitHub commits.
// It contains a pointer to a GitHub client from the go-github library.
type GitHubCommitsOpsClient struct {
	GitHubClient *github.Client
}

// NewGitHubCommitsOpsClient creates a new GitHubCommitsOpsClient with the given http.Client.
// It initializes the GitHubClient inside GitHubCommitsOpsClient using the provided http.Client.
// The GitHubClient is responsible for interacting with the GitHub API.
func NewGitHubCommitsOpsClient(httpClient *http.Client) *GitHubCommitsOpsClient {
	gc := github.NewClient(httpClient)
	return &GitHubCommitsOpsClient{GitHubClient: gc}
}

// ListCommits fetches the list of commits for a specific repository.
func (gClient *GitHubCommitsOpsClient) ListCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*github.RepositoryCommit, *github.Response, error) {
	return gClient.GitHubClient.Repositories.ListCommits(ctx, owner, repo, opts)
}

// GetCommit retrieves a specific commit from a repository.
//
// ctx is the context.Context used for the API call.
//
// owner is the username or organization name of the repository owner.
//
// repo is the name of the repository.
//
// sha is the SHA of the commit.
//
// opts specifies optional parameters for the API call.
//
// The function returns the retrieved commit information as a *github.RepositoryCommit,
// the HTTP response as *github.Response, and an error if any.
func (gClient *GitHubCommitsOpsClient) GetCommit(ctx context.Context, owner, repo, sha string, opts *github.ListOptions) (*github.RepositoryCommit, *github.Response, error) {
	return gClient.GitHubClient.Repositories.GetCommit(ctx, owner, repo, sha, opts)
}

// GHQueryForListFiles is a struct representing the GraphQL query for listing files in a GitHub repository.
// It contains the information necessary to make the query, including the owner, name, expression, and path of the repository.
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

// NewGitHubClient creates a new instance of the GitHub client.
// It takes a CommitOpsClient, GraphQLClient, and GitHubConfig as parameters and returns a pointer to a GitHub struct.
// The CommitOpsClient is responsible for making REST API calls to the GitHub API.
// The GraphQLClient is responsible for making GraphQL API calls to the GitHub API.
// The GitHubConfig contains the configuration parameters for the GitHub client, such as owner, repositories, default branch,
// and filter options.
// The new GitHub client is initialized with the provided CommitOpsClient, GraphQLClient, and GitHubConfig.
// The GitHub client can be used to interact with the GitHub API and perform various operations, such as retrieving file paths for repositories
// and getting changed file paths since a specified time.
// Usage example:
// ```
// commitOpsClient := NewGitHubCommitsOpsClient()
// graphQLClient := NewGraphQLClient()
//
//	config := GitHubConfig{
//	    Owner:         "testowner",
//	    Repositories:  []string{"repo1", "repo2"},
//	    DefaultBranch: "main",
//	    Filter: GitHubFilter{
//	        FilePath:  "path/to/files",
//	        FileTypes: []string{".txt"},
//	    },
//	}
//
// githubClient := NewGitHubClient(commitOpsClient, graphQLClient, config)
// filepaths, err := githubClient.GetFilePathsForRepositories()
//
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, path := range filepaths {
//	    fmt.Println(path)
//	}
//
// ```
func NewGitHubClient(commitOpsClient CommitOpsClient, graphQLClient GraphQLClient, configuration GitHubConfig) *GitHub {
	return &GitHub{
		commitOpsClient: commitOpsClient,
		graphQLClient:   graphQLClient,
		Configuration:   configuration,
	}
}

// GetFilePathsForRepositories retrieves the file paths for the repositories specified in the GitHub configuration.
// It iterates over each repository, calls the getFilePathsForRepo method to get the file paths, and appends them to the files slice.
// If there are no file types specified in the configuration, it returns the files directly.
// Otherwise, it filters the files based on the file types specified in the configuration and returns the filtered files.
// If there's an error during the process, it returns nil and the error.
//
// Usage:
// ```
//
//	repos := []string{"repo1", "repo2", "repo3"}
//	filePaths, err := GetFilePathsForRepositories(repos)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, paths := range filePaths {
//	    for _, path := range paths {
//	        fmt.Println(path)
//	    }
//	}
//
// ```
func (c *GitHub) GetFilePathsForRepositories() ([]string, error) {
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

// GetChangedFilePathsSince retrieves the list of file paths that have changed in the specified repositories within the specified time frame.
// It takes the number of hours as input and returns a Paths object containing the lists of added, removed, and modified file paths.
// The function iterates over the repositories defined in the GitHub configuration and calls the getChangedFilePathsForRepo function to get the file paths for each repository.
// It then aggregates the file paths from all repositories into a single Paths object and returns it.
// The function uses the specified time frame and file path filter defined in the GitHub configuration to fetch the changed file paths.
// It uses the GitHub commit operations client to fetch the commits and commit details for each repository.
// Finally, it filters the file paths based on the directory filter and populates the added, removed, and modified lists in the Paths object accordingly.
//
// Parameters:
// - hoursSince: The number of hours since the specified time to consider for fetching changed file paths.
//
// Returns:
// - Paths: A Paths object containing the lists of added, removed, and modified file paths.
// - error: An error, if any occurred during the execution of the function.
//
// Usage:
// ```
//
//	const sinceHours = "24"
//	changedFiles, err := GetChangedFilePathsSince(sinceHours)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// fmt.Println(file.Added)
// fmt.Println(file.Modified)
// fmt.Println(file.Removed)
// ```
func (c *GitHub) GetChangedFilePathsSince(hoursSince int) (Paths, error) {
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

// getFilePathsForRepo fetches the list of file paths for a specific repository, starting from the specified expression.
// It recursively traverses the repository tree, appending file paths to the resulting slice, and returns the slice of file paths.
// If any error occurs during the GraphQL query or traversal, it returns nil and the error.
// The owner parameter specifies the repository owner's username.
// The name parameter specifies the repository name.
// The expression parameter specifies the starting expression for traversing the repository tree.
// It uses the GitHub GraphQL API to retrieve the repository tree entries and their types.
// If an entry is a blob, its path is appended to the files slice.
// If an entry is a tree, the function recursively calls itself with the updated expression and appends the returned subfiles to the files slice.
// Returns:
// - files: The slice of file paths in the repository.
// - error: Any error that occurred during the GraphQL query or traversal.
func (c *GitHub) getFilePathsForRepo(owner, name, expression string) ([]string, error) {
	var query GHQueryForListFiles
	variables := map[string]interface{}{
		"owner":      githubv4.String(owner),
		"name":       githubv4.String(name),
		"expression": githubv4.String(expression),
	}

	err := c.graphQLClient.Query(context.Background(), &query, variables)
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

// hasFileType checks if the given fileName ends with any of the fileTypes.
func (c *GitHub) hasFileType(fileName string, fileTypes []string) bool {
	for _, fileType := range fileTypes {
		if strings.HasSuffix(fileName, fileType) {
			return true
		}
	}
	return false
}

// getChangedFilePathsForRepo fetches the paths of files that have been changed in a specific repository.
// It takes the repository name, a CommitsListOptions object for filtering commits, and returns a Paths struct with added, removed, and modified files.
// The method iterates through the commits in the repository, retrieves commit details, and checks each file in the commit against the filter path.
// Depending on the type of change (added, removed, modified, renamed, copied), the file path is appended to the respective list in the Paths struct.
// The method returns the Paths struct and an error, if any.
func (c *GitHub) getChangedFilePathsForRepo(ctx context.Context, repo string, opt *github.CommitsListOptions) (Paths, error) {
	var paths Paths

	commits, _, err := c.commitOpsClient.ListCommits(ctx, c.Configuration.Owner, repo, opt)
	if err != nil {
		return paths, err
	}

	directory := c.Configuration.Filter.FilePath

	for _, commit := range commits {
		commitDetails, _, err := c.commitOpsClient.GetCommit(ctx, c.Configuration.Owner, repo, *commit.SHA, nil)
		if err != nil {
			return paths, err
		}

		for _, file := range commitDetails.Files {
			if strings.HasPrefix(file.GetFilename(), directory) {
				switch file.GetStatus() {
				case "removed":
					paths.Removed = append(paths.Removed, file.GetFilename())
				case "added":
					paths.Added = append(paths.Added, file.GetFilename())
				case "modified", "changed":
					paths.Modified = append(paths.Modified, file.GetFilename())
				case "renamed":
					paths.Removed = append(paths.Removed, file.GetPreviousFilename())
					paths.Added = append(paths.Added, file.GetFilename())
				case "copied":
					paths.Added = append(paths.Added, file.GetFilename())
				}
			}
		}
	}

	return paths, nil
}
