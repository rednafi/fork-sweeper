package src

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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

var httpClient = &http.Client{Timeout: 10 * time.Second}

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

	req.Header.Add("Authorization", "Bearer "+token)

	var repos []repo
	if err := doRequest(req, &repos); err != nil {
		return nil, err
	}

	var forkedRepos []repo
	for _, repo := range repos {
		if repo.IsFork {
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
	maxPage int) ([]repo, error) {

	var allRepos []repo
	for pageNum := 1; pageNum <= maxPage; pageNum++ {
		repos, err := fetchForkedReposPage(ctx, baseURL, owner, token, pageNum, perPage)
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
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
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

func CLI() {
	var (
		owner   string
		token   string
		perPage int
		maxPage int
	)

	// Parsing command-line flags
	flag.StringVar(&owner, "owner", "", "GitHub repository owner (required)")
	flag.StringVar(&token, "token", "", "GitHub personal access token (required)")
	flag.IntVar(&perPage, "per-page", 100, "Number of repositories per page")
	flag.IntVar(&maxPage, "max-page", 100, "Maximum page number to fetch")
	flag.Parse()

	// Validating required arguments
	if owner == "" || token == "" {
		fmt.Fprintf(os.Stderr, "%sError:%s Owner and token are required.\n", Red, Reset)
		flag.PrintDefaults()
		os.Exit(1)
	}

	ctx := context.Background()
	baseURL := "https://api.github.com"

	// Fetching repositories
	printWithColor(Blue, fmt.Sprintf("\nFetching repositories for %s...\n", owner))
	forkedRepos, err := fetchForkedRepos(ctx, baseURL, owner, token, perPage, maxPage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError fetching repositories:%s %v\n", Red, Reset, err)
		os.Exit(1)
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
	printWithColor(Blue, "\nDeleting forked repositories...\n")
	if err := deleteRepos(ctx, baseURL, token, forkedRepos); err != nil {
		fmt.Fprintf(os.Stderr, "%sError deleting repositories:%s %v\n", Red, Reset, err)
		os.Exit(1)
	}
	printWithColor(Green, "Deletion completed successfully.")
}
