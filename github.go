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
// Parameters:
//   - ctx: The context.Context used for the API call. It allows you to cancel
//     the request, set deadlines, etc.
//   - owner: The username or organization name of the repository owner. This
//     string identifies the owner of the repository.
//   - repo: The name of the repository. It specifies which repository's commit
//     is being retrieved.
//   - sha: The SHA hash of the commit. This string uniquely identifies the commit
//     within the repository.
//   - opts: Optional parameters for the API call, provided as a pointer to
//     github.ListOptions. This includes pagination options.
//
// Returns:
//   - *github.RepositoryCommit: The retrieved commit information, including details
//     like the commit message, author, etc.
//   - *github.Response: The HTTP response from the API call. This includes
//     information like the status code and headers.
//   - error: An error instance if an error occurs during the API call. It will be
//     nil if the call is successful.
//
// Example:
//
//	commit, resp, err := gClient.GetCommit(ctx, "octocat", "hello-world", "6dcb09b5b57875f334f61aebed695e2e4193db5e", nil)
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
// Usage:
//
//	 commitOpsClient := NewGitHubCommitsOpsClient()
//	 graphQLClient := NewGraphQLClient()
//
//		config := GitHubConfig{
//		    Owner:         "testowner",
//		    Repositories:  []string{"repo1", "repo2"},
//		    DefaultBranch: "main",
//		    Filter: GitHubFilter{
//		        FilePath:  "path/to/files",
//		        FileTypes: []string{".txt"},
//		    },
//		}
//
//	 githubClient := NewGitHubClient(commitOpsClient, graphQLClient, config)
//	 filepaths, err := githubClient.GetFilePathsForRepositories()
//
//		if err != nil {
//		    log.Fatal(err)
//		}
//
//		for _, path := range filepaths {
//		    fmt.Println(path)
//		}
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

// GetChangedFilePathsSince retrieves the list of file paths that have changed in the specified repositories
// within the specified time frame. The function iterates over repositories defined in the GitHub configuration
// and uses the GitHub commit operations client to fetch the commits and commit details for each repository.
// It aggregates the file paths from all repositories into a single Paths object. The function filters these file
// paths based on the directory filter and the specified time frame and file path filter defined in the GitHub
// configuration. The Paths object is populated with lists of added, removed, and modified file paths accordingly.
//
// Parameters:
//   - hoursSince: An integer representing the number of hours since the specified time. This parameter is used
//     to determine the time frame for fetching changed file paths.
//
// Returns:
//   - Paths: A struct containing lists of added, removed, and modified file paths. This struct provides an
//     organized way to access the changed files.
//   - error: An error instance if an error occurs during the execution of the function. It will be nil if
//     the function executes successfully.
//
// Usage:
//
//	const sinceHours = 24
//	changedFiles, err := c.GetChangedFilePathsSince(sinceHours)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println("Added files:", changedFiles.Added)
//	fmt.Println("Modified files:", changedFiles.Modified)
//	fmt.Println("Removed files:", changedFiles.Removed)
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

// getFilePathsForRepo fetches the list of file paths for a specific repository, starting from the specified
// expression. It uses the GitHub GraphQL API to retrieve the repository tree entries and their types, and
// recursively traverses the repository tree. The function appends file paths to a slice, which is then returned.
// If an entry is a blob, its path is added to the files slice. For tree entries, the function recurses with
// the updated expression and appends the returned subfiles to the files slice. If any error occurs during
// the GraphQL query or traversal, the function returns nil and the error.
//
// Parameters:
//   - owner: A string representing the username of the repository owner. This parameter specifies the owner
//     of the repository for which file paths are being fetched.
//   - name: A string representing the name of the repository. This parameter is used to specify the repository
//     from which the file paths are retrieved.
//   - expression: A string specifying the starting expression for traversing the repository tree. This
//     expression determines the starting point of the file path retrieval process.
//
// Returns:
//   - files: A slice of strings, each representing a file path in the repository. This slice includes paths
//     to all files found in the repository starting from the given expression.
//   - error: An error instance, if any error occurred during the GraphQL query or traversal. It will be nil
//     if the function executes successfully.
//
// Example usage:
//
//	filePaths, err := c.getFilePathsForRepo("octocat", "hello-world", "master:")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, path := range filePaths {
//	    fmt.Println(path)
//	}
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
