package src

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	exitOk  = 0
	exitErr = 1

	errUserNotFound                = "API request failed with status: 404"
	errInvalidToken                = "API request failed with status: 401"
	errInsufficientTokenPermission = "API request failed with status: 403"
)

type repo struct {
	Name   string `json:"full_name"`
	URL    string `json:"html_url"`
	IsFork bool   `json:"fork"`
	Owner  struct {
		Name string `json:"name"`
	} `json:"owner"`
	CreatedAt time.Time `json:"created_at"`
}

var httpClientPool = sync.Pool{
	New: func() any {
		return &http.Client{Timeout: 10 * time.Second}
	},
}

func fetchForkedReposPage(
	ctx context.Context,
	baseURL,
	owner,
	token string,
	pageNum,
	perPage,
	olderThanDays int) ([]repo, error) {

	url := fmt.Sprintf(
		"%s/users/%s/repos?type=forks&page=%d&per_page=%d",
		baseURL,
		owner,
		pageNum,
		perPage)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+token)

	var repos []repo
	if err := doRequest(req, &repos); err != nil {
		return nil, err
	}

	var forkedRepos []repo

	cutOffDate := time.Now().AddDate(0, 0, -olderThanDays)

	for _, repo := range repos {
		if repo.IsFork && repo.CreatedAt.Before(cutOffDate) {
			forkedRepos = append(forkedRepos, repo)
		}
	}

	return forkedRepos, nil
}

func fetchForkedRepos(
	ctx context.Context,
	baseURL,
	owner,
	token string,
	perPage,
	maxPage,
	olderThanDays int) ([]repo, error) {

	var allRepos []repo
	for pageNum := 1; pageNum <= maxPage; pageNum++ {
		repos, err := fetchForkedReposPage(
			ctx,
			baseURL,
			owner,
			token,
			pageNum,
			perPage,
			olderThanDays)

		if err != nil {
			return nil, err
		}

		if len(repos) == 0 {
			break
		}

		allRepos = append(allRepos, repos...)
	}
	return allRepos, nil
}

func doRequest(req *http.Request, v any) error {
	httpClient := httpClientPool.Get().(*http.Client)
	defer httpClientPool.Put(httpClient)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return err
		}
	}
	return nil
}

func deleteRepo(ctx context.Context, baseURL, owner, name, token string) error {
	url := fmt.Sprintf("%s/repos%s/%s", baseURL, owner, name)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	return doRequest(req, nil)
}

func deleteRepos(ctx context.Context, baseURL, token string, repos []repo) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	for _, r := range repos {
		wg.Add(1)
		go func(r repo) {
			defer wg.Done()
			if err := deleteRepo(ctx, baseURL, r.Owner.Name, r.Name, token); err != nil {
				select {
				case errChan <- err:
				default:
				}
			}
		}(r)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return <-errChan
	}
	return nil
}

type cliConfig struct {
	// Required
	stdout  io.Writer
	stderr  io.Writer
	version string

	// Optional
	flagErrorHandling flag.ErrorHandling
	fetchForkedRepos  func(
		ctx context.Context,
		baseURL,
		owner,
		token string,
		perPage,
		maxPage,
		olderThanDays int) ([]repo, error)
	deleteRepos func(ctx context.Context, baseURL, token string, repos []repo) error
}

func NewCLIConfig(
	stdout,
	stderr io.Writer,
	version string,
) *cliConfig {

	return &cliConfig{
		stdout:  stdout,
		stderr:  stderr,
		version: version,

		flagErrorHandling: flag.ExitOnError,
		fetchForkedRepos:  fetchForkedRepos,
		deleteRepos:       deleteRepos,
	}
}

// Dysfunctional options pattern
func (c *cliConfig) withFlagErrorHandling(h flag.ErrorHandling) *cliConfig {
	c.flagErrorHandling = h
	return c
}

func (c *cliConfig) withFetchForkedRepos(
	f func(
		ctx context.Context,
		baseURL,
		owner,
		token string,
		perPage,
		maxPage,
		olderThanDays int) ([]repo, error)) *cliConfig {

	c.fetchForkedRepos = f
	return c
}

func (c *cliConfig) withDeleteRepos(
	f func(ctx context.Context, baseURL, token string, repos []repo) error) *cliConfig {

	c.deleteRepos = f
	return c
}

func (c *cliConfig) CLI(args []string) int {
	var (
		owner         string
		token         string
		perPage       int
		maxPage       int
		olderThanDays int
		version       bool
		delete        bool

		stdout            = c.stdout
		stderr            = c.stderr
		versionNum        = c.version
		flagErrorHandling = c.flagErrorHandling
		fetchForkedRepos  = c.fetchForkedRepos
		deleteRepos       = c.deleteRepos
	)

	// Parsing command-line flags
	fs := flag.NewFlagSet("fork-sweeper", flagErrorHandling)
	fs.SetOutput(stdout)

	fs.StringVar(&owner, "owner", "", "GitHub repo owner (required)")
	fs.StringVar(&token, "token", "", "GitHub access token (required)")
	fs.IntVar(&perPage, "per-page", 100, "Number of forked repos fetched per page")
	fs.IntVar(&maxPage, "max-page", 100, "Maximum number of pages to fetch")
	fs.IntVar(
		&olderThanDays,
		"older-than-days",
		60,
		"Fetch forked repos older than this number of days")
	fs.BoolVar(&version, "version", false, "Print version")
	fs.BoolVar(&delete, "delete", false, "Delete forked repos")

	fs.Parse(args)

	// Printing version
	if version {
		fmt.Fprintln(stdout, versionNum)
		return exitOk
	}

	// Validating required arguments
	if owner == "" || token == "" {
		fmt.Fprintln(stderr, "Error: owner and token are required")
		fs.PrintDefaults()
		return exitErr
	}

	ctx := context.Background()
	baseURL := "https://api.github.com"

	// Fetching repositories
	fmt.Fprintf(stdout, "\nFetching repositories for %s...\n", owner)
	forkedRepos, err := fetchForkedRepos(
		ctx,
		baseURL,
		owner,
		token,
		perPage,
		maxPage,
		olderThanDays)

	if err != nil {
		switch err.Error() {
		case errUserNotFound:
			fmt.Fprintf(stderr, "Error: user not found\n")
		case errInvalidToken:
			fmt.Fprintf(stderr, "Error: invalid token\n")
		default:
			fmt.Fprintf(stderr, "Error: %s\n", err)
		}
		return exitErr
	}
	if len(forkedRepos) == 0 {
		fmt.Fprintf(stdout, "\nNo forked repositories found\n")
		return exitOk
	}

	// Listing forked repositories
	fmt.Fprintf(stdout, "\nForked repos:\n")
	for _, repo := range forkedRepos {
		fmt.Fprintf(stdout, "    - %s\n", repo.URL)
	}

	// Deleting forked repositories
	if !delete {
		return exitOk
	}

	fmt.Fprintf(stdout, "\nDeleting forked repositories...\n")
	if err := deleteRepos(ctx, baseURL, token, forkedRepos); err != nil {
		switch err.Error() {
		case errInsufficientTokenPermission:
			fmt.Fprintf(stderr, "Error: token does not have permission to delete repos\n")
		default:
			fmt.Fprintf(stderr, "Error: %s\n", err)
		}
		return exitErr
	}

	fmt.Fprintf(stdout, "\nForks deleted successfully\n")
	return exitOk
}
