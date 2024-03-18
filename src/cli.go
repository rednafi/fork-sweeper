package src

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// Exit codes
	exitOk  = 0
	exitErr = 1

	// Error messages to catch from the GitHub API
	ErrMsg401 = "API request failed with status: 401"
	ErrMsg403 = "API request failed with status: 403"
	ErrMsg404 = "API request failed with status: 404"
)

type repo struct {
	Name   string `json:"name"`
	URL    string `json:"html_url"`
	IsFork bool   `json:"fork"`
	Owner  struct {
		Name string `json:"login"`
	} `json:"owner"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	PushedAt  time.Time `json:"pushed_at"`
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
	perPage int) ([]repo, error) {

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

	var repos []repo
	if err := doRequest(req, token, &repos); err != nil {
		return nil, err
	}

	// Filter out non-forked repositories
	var forkedRepos []repo
	for _, r := range repos {
		if r.IsFork {
			forkedRepos = append(forkedRepos, r)
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
	maxPage int) ([]repo, error) {

	var allRepos []repo
	for pageNum := 1; pageNum <= maxPage; pageNum++ {
		repos, err := fetchForkedReposPage(
			ctx,     // ctx
			baseURL, // baseURL
			owner,   // owner
			token,   // token
			pageNum, // pageNum
			perPage, // perPage
		)

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

func doRequest(req *http.Request, token string, result any) error {
	httpClient := httpClientPool.Get().(*http.Client)
	defer httpClientPool.Put(httpClient)

	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	req.Header.Add("User-Agent", "Mozilla/5.0")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return err
		}
	}
	return nil
}

// filterForkedRepos filters forked repositories based on their update date and whether their name matches any in the protectedRepos list using a basic form of fuzzy matching.
func filterForkedRepos(
	forkedRepos []repo,
	guardedRepoNames []string,
	olderThanDays int) ([]repo, []repo) {

	unguardedRepos, guardedRepos := []repo{}, []repo{}

	now := time.Now()

	// Convert olderThanDays to duration and subtract from current time to get cutoff date
	cutOffDate := now.Add(time.Duration(-olderThanDays) * 24 * time.Hour)

	for _, repo := range forkedRepos {
		// Check if repo activity is after cutoff date or name matches guarded list
		hasRecentActivity := repo.PushedAt.After(cutOffDate) ||
			repo.UpdatedAt.After(cutOffDate) || repo.CreatedAt.After(cutOffDate)

		isGuardedName := false
		for _, name := range guardedRepoNames {
			repoName := strings.ToLower(repo.Name)
			name = strings.ToLower(name)

			if strings.TrimSpace(name) != "" && strings.Contains(repoName, name) {
				isGuardedName = true
				break
			}
		}

		if hasRecentActivity || isGuardedName {
			guardedRepos = append(guardedRepos, repo)
		} else {
			unguardedRepos = append(unguardedRepos, repo)
		}
	}

	return unguardedRepos, guardedRepos
}

func deleteRepo(ctx context.Context, baseURL, owner, name, token string) error {
	url := fmt.Sprintf("%s/repos/%s/%s", baseURL, owner, name)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	return doRequest(req, token, nil)
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
		maxPage int) ([]repo, error)

	filterForkedRepos func(
		forkedRepos []repo,
		protectedRepos []string,
		olderThanDays int) ([]repo, []repo)

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
		maxPage int) ([]repo, error)) *cliConfig {

	c.fetchForkedRepos = f
	return c
}

func (c *cliConfig) withFilterForkedRepos(
	f func(
		forkedRepos []repo,
		protectedRepos []string,
		olderThanDays int) ([]repo, []repo)) *cliConfig {

	c.filterForkedRepos = f
	return c
}

func (c *cliConfig) withDeleteRepos(
	f func(ctx context.Context, baseURL, token string, repos []repo) error) *cliConfig {

	c.deleteRepos = f
	return c
}

type stringSlice []string

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *stringSlice) String() string {
	return strings.Join(*s, ", ")
}

func (c *cliConfig) CLI(args []string) int {
	var (
		owner          string
		token          string
		perPage        int
		maxPage        int
		olderThanDays  int
		version        bool
		delete         bool
		protectedRepos stringSlice

		stdout            = c.stdout
		stderr            = c.stderr
		versionNumber     = c.version
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
	fs.IntVar(&olderThanDays,
		"older-than-days",
		60,
		"Fetch forked repos modified more than n days ago")
	fs.BoolVar(&version, "version", false, "Print version")
	fs.BoolVar(&delete, "delete", false, "Delete forked repos")
	fs.Var(&protectedRepos, "guard", "List of repos to protect from deletion (fuzzy match name)")

	fs.Parse(args)

	// Printing version
	if version {
		fmt.Fprintln(stdout, versionNumber)
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
	fmt.Fprintf(stdout, "\nFetching forked repositories for %s...\n", owner)
	forkedRepos, err := fetchForkedRepos(
		ctx,     // ctx
		baseURL, // baseURL
		owner,   // owner
		token,   // token
		perPage, // perPage
		maxPage, // maxPage
	)

	if err != nil {
		switch err.Error() {
		case ErrMsg404:
			fmt.Fprintf(stderr, "Error: user not found\n")
		case ErrMsg401:
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

	// Filtering repositories
	unguardedRepos, guardedRepos := filterForkedRepos(
		forkedRepos,
		protectedRepos,
		olderThanDays)

	// Displaying safeguarded repositories
	fmt.Fprintf(stdout, "\nGuarded forked repos [won't be deleted]:\n")
	for _, repo := range guardedRepos {
		fmt.Fprintf(stdout, "    - %s\n", repo.URL)
	}

	// Displaying unguarded repositories
	fmt.Fprintf(stdout, "\nUnguarded forked repos [will be deleted]:\n")
	for _, repo := range unguardedRepos {
		fmt.Fprintf(stdout, "    - %s\n", repo.URL)
	}

	// Deleting unguarded repositories
	if !delete {
		return exitOk
	}

	if len(unguardedRepos) == 0 {
		fmt.Fprintf(stdout, "\nNo unguarded forked repositories to delete\n")
		return exitOk
	}

	fmt.Fprintf(stdout, "\nDeleting forked repositories...\n")
	if err := deleteRepos(ctx, baseURL, token, unguardedRepos); err != nil {
		switch err.Error() {
		case ErrMsg403:
			fmt.Fprintf(stderr, "Error: token does not have permission to delete repos\n")
		case ErrMsg404:
			fmt.Fprintf(stderr, "Error: repo not found\n")
		default:
			fmt.Fprintf(stderr, "Error: %s\n", err)
		}
		return exitErr
	}

	fmt.Fprintf(stdout, "\nForks deleted successfully\n")
	return exitOk
}
