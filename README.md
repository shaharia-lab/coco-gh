# coco-gh

`coco-gh` is a Go library dedicated to collecting GitHub content.

## Features

- Fetch all file paths based on the configuration.
- Fetch a list of file paths that were changed in the last `X` hours.

## Getting Started

Download the library using `go get`:

```bash
go get github.com/shaharia-lab/coco-gh
```

Import the library in your code:

```go
import "github.com/shaharia-lab/coco-gh"
```

### Usage

```go
package cocogh

import (
	"context"
	"log"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func main() {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	ghCommitsOpsClient := NewGitHubCommitsOpsClient(httpClient)
	graphQLClient := githubv4.NewClient(httpClient)
	ghConfig := GitHubConfig{
		Owner:         "kubernetes",
		Repositories:  []string{"website"},
		DefaultBranch: "main",
		Filter: GitHubFilter{
			FilePath: "content/en/blog/_posts",
			FileTypes: []string{
				".md",
			},
		},
	}

	ch := NewGitHubClient(ghCommitsOpsClient, graphQLClient, ghConfig)
	
	// Get all the file paths from the repositories
	allFilePaths, err := ch.GetFilePathsForRepositories()
	if err != nil {
		// handle errors
	}

	for _, path := range allFilePaths {
		log.Println(path)
	}
	
	// Get the list of files that were changed in the last X hours
	contentChanged, err := ch.GetChangedFilePathsSince(24)
	if err != nil {
		// handle errors
	}

	log.Println(contentChanged.Added)
	log.Println(contentChanged.Modified)
	log.Println(contentChanged.Removed)
}
```

## Contributing

Contributions to [coco-gh](https://github.com/shaharia-lab/coco-gh) are more than welcome! If you're looking to contribute to our project, you're in the right place. Here are some ways you can help:

1. **Report Bugs**: If you find a bug, please open an issue to report it. Describe the bug, how to reproduce it, and the environment (e.g., OS, Go version).

2. **Suggest Enhancements**: Have an idea to make this project better? Open an issue to suggest your idea. Whether it's a new feature, code improvement, or documentation updates, we'd love to hear from you.

3. **Submit Pull Requests**: Feel free to fork the repository and submit pull requests. Before submitting your pull request, please ensure the following:
    - Your code follows the project's coding standards.
    - All tests are passing.
    - Add or update tests as necessary for your code.
    - Update the documentation to reflect your changes, if applicable.
    - Include a clear description in your PR about the changes you have made.

4. **Review Pull Requests**: If you're interested in contributing by reviewing pull requests, please feel free to do so. Any feedback or suggestions are highly valuable.

We appreciate your contributions and look forward to your active participation in the development of [coco-gh](https://github.com/shaharia-lab/coco-gh)!

## License
[coco-gh](https://github.com/shaharia-lab/coco-gh)  is licensed under the MIT License.