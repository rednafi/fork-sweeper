package src

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Reset  = "\033[0m"
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

	for _, repo := range repos {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := deleteRepo(ctx, baseURL, repo.Owner.Name, repo.Name, token); err != nil {
				select {
				case errChan <- err:
				default:
				}
			}
		}()
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return <-errChan
	}
	return nil
}

func printWithColor(color, text string) {
	fmt.Println(color + text + Reset)
}

type CLIConfig struct {

	// Required
	writer   io.Writer
	version  string
	exitFunc func(int)

	// Optional
	flagErrorHandling flag.ErrorHandling
	printWithColor    func(color, text string)
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

// Dysfunctional options pattern

func (c *CLIConfig) WithFlagErrorHandling(h flag.ErrorHandling) *CLIConfig {
	c.flagErrorHandling = h
	return c
}

func (c *CLIConfig) WithPrintWithColor(f func(color, text string)) *CLIConfig {
	c.printWithColor = printWithColor
	return c
}

func (c *CLIConfig) WithFetchForkedRepos(
	f func(
		ctx context.Context,
		baseURL,
		owner,
		token string,
		perPage,
		maxPage,
		olderThanDays int) ([]repo, error)) *CLIConfig {

	c.fetchForkedRepos = f
	return c
}

func (c *CLIConfig) WithDeleteRepos(
	f func(ctx context.Context, baseURL, token string, repos []repo) error) *CLIConfig {

	c.deleteRepos = f
	return c
}

func NewCLIConfig(
	writer io.Writer,
	version string,
	exitFunc func(int),
) *CLIConfig {

	return &CLIConfig{
		writer:            writer,
		version:           version,
		exitFunc:          exitFunc,
		flagErrorHandling: flag.ExitOnError,
		printWithColor:    printWithColor,
		fetchForkedRepos:  fetchForkedRepos,
		deleteRepos:       deleteRepos,
	}
}

func (c *CLIConfig) CLI(args []string) {
	var (
		owner             string
		token             string
		perPage           int
		maxPage           int
		olderThanDays     int
		version           bool
		delete            bool
		writer            = c.writer
		versionNum        = c.version
		exitFunc          = c.exitFunc
		flagErrorHandling = c.flagErrorHandling
		printWithColor    = c.printWithColor
		fetchForkedRepos  = c.fetchForkedRepos
		deleteRepos       = c.deleteRepos
	)

	// Parsing command-line flags
	fs := flag.NewFlagSet("fork-sweeper", flagErrorHandling)
	fs.SetOutput(writer)

	fs.StringVar(&owner, "owner", "", "GitHub repo owner (required)")
	fs.StringVar(&token, "token", "", "GitHub access token (required)")
	fs.IntVar(&perPage, "per-page", 100, "Number of forked repos fetched per page")
	fs.IntVar(&maxPage, "max-page", 100, "Maximum page number to fetch")
	fs.IntVar(
		&olderThanDays,
		"older-than-days",
		60,
		"Delete forked repos older than this number of days")
	fs.BoolVar(&version, "version", false, "Print version")
	fs.BoolVar(&delete, "delete", false, "Delete forked repos")
	fs.Parse(args)

	// Printing version
	if version {
		fmt.Println(versionNum)
		return
	}

	// Validating required arguments
	if owner == "" || token == "" {
		fmt.Fprintf(os.Stderr, "%sError:%s Owner and token are required.\n", Red, Reset)
		fs.PrintDefaults()
		exitFunc(1)
	}

	ctx := context.Background()
	baseURL := "https://api.github.com"

	// Fetching repositories
	printWithColor(Blue, fmt.Sprintf("\nFetching repositories for %s...\n", owner))
	forkedRepos, err := fetchForkedRepos(
		ctx,
		baseURL,
		owner,
		token,
		perPage,
		maxPage,
		olderThanDays)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError fetching repositories:%s %v\n", Red, Reset, err)
		exitFunc(1)
	}

	if len(forkedRepos) == 0 {
		printWithColor(Green, "No forked repositories found.")
		return
	}

	// Listing forked repositories
	printWithColor(Blue, "Forked repos:\n")
	for _, repo := range forkedRepos {
		fmt.Println("  - ", repo.Name)
	}

	// Deleting forked repositories
	if !delete {
		return
	}

	printWithColor(Blue, "\nDeleting forked repositories...\n")
	if err := deleteRepos(ctx, baseURL, token, forkedRepos); err != nil {
		fmt.Fprintf(os.Stderr, "%sError deleting repositories:%s %v\n", Red, Reset, err)
		exitFunc(1)
	}
	printWithColor(Green, "Deletion completed successfully.")
}
