package src

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestUnmarshalRepo tests the unmarshalling of a JSON string into a repo struct
func TestUnmarshalRepo(t *testing.T) {
	t.Parallel()
	// Example JSON string that represents a repo's data
	jsonStr := `{
		"name": "test-repo",
		"html_url": "https://github.com/test-owner/test-repo",
		"fork": false,
		"owner": {
			"login": "test-owner"
		},
		"created_at": "2020-01-01T00:00:00Z",
		"updated_at": "2020-01-01T00:00:00Z",
		"pushed_at": "2020-01-01T00:00:00Z"
	}`

	// Expected repo object based on the JSON string
	expected := repo{
		Name:   "test-repo",
		URL:    "https://github.com/test-owner/test-repo",
		IsFork: false,
		Owner: struct {
			Name string `json:"login"`
		}{
			Name: "test-owner",
		},
		CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		PushedAt:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Unmarshal the JSON string into a repo struct
	var result repo
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		t.Fatalf("Unmarshalling failed: %v", err)
	}

	// Compare the expected and actual repo structs
	if !reflect.DeepEqual(expected, result) {
		t.Errorf(
			`Unmarshalled repo does not match expected value.
			Expected %+v, got %+v`, expected, result)
	}
}

// TestFetchForkedReposPage with adjusted repo struct
func TestFetchForkedReposPage(t *testing.T) {
	t.Parallel()
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(
				w,
				`[{"name": "test-forked-repo",`+
					`"html_url": "https://github.com/test-owner/test-forked-repo", `+
					`"fork": true,`+
					`"owner": {"login": "test-owner"},`+
					`"created_at": "2020-01-01T00:00:00Z",`+
					`"updated_at": "2020-01-01T00:00:00Z",`+
					`"pushed_at": "2020-01-01T00:00:00Z"}]`)
		}))
	defer mockServer.Close()

	expected := []repo{
		{
			Name:   "test-forked-repo",
			URL:    "https://github.com/test-owner/test-forked-repo",
			IsFork: true,
			Owner: struct {
				Name string `json:"login"`
			}{Name: "test-owner"},
			CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			PushedAt:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	forkedRepos, err := fetchForkedReposPage(
		context.Background(), // ctx
		mockServer.URL,       // baseURL
		"test-owner",         // owner
		"test-token",         // token
		1,                    // pageNum
		10,                   // perPage
	)

	if err != nil {
		t.Fatalf("fetchForkedReposPage returned an error: %v", err)
	}

	if len(forkedRepos) != len(expected) {
		t.Fatalf("Expected %d forked repos, got %d", len(expected), len(forkedRepos))
	}

	for i, repo := range forkedRepos {
		if repo.Name != expected[i].Name ||
			repo.URL != expected[i].URL ||
			repo.IsFork != expected[i].IsFork ||
			repo.Owner.Name != expected[i].Owner.Name ||
			!repo.UpdatedAt.Equal(expected[i].UpdatedAt) {
			t.Errorf("Expected repo %+v, got %+v", expected[i], repo)
		}
	}
}

func TestFetchForkedRepos(t *testing.T) {
	t.Parallel()
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(
				w,
				`[{"name": "test-repo-1",`+
					`"html_url": "https://test.com/test-owner/test-repo-1",`+
					`"fork": true,`+
					`"owner": {"login": "test-owner"},`+
					`"created_at": "2020-01-01T00:00:00Z",`+
					`"updated_at": "2020-01-01T00:00:00Z",`+
					`"pushed_at": "2020-01-01T00:00:00Z"},`+

					`{"name": "test-repo-2",`+
					`"html_url": "https://test.com/test-owner/test-repo-2",`+
					`"fork": true,`+
					`"owner": {"login": "test-owner"},`+
					`"created_at": "2020-01-01T00:00:00Z",`+
					`"updated_at": "2020-01-01T00:00:00Z",`+
					`"pushed_at": "2020-01-01T00:00:00Z"}]`)
		}))

	defer mockServer.Close()

	expected := []repo{
		{
			Name:   "test-repo-1",
			URL:    "https://test.com/test-owner/test-repo-1",
			IsFork: true,
			Owner: struct {
				Name string `json:"login"`
			}{Name: "test-owner"},
			CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			PushedAt:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Name:   "test-repo-2",
			URL:    "https://test.com/test-owner/test-repo-2",
			IsFork: true,
			Owner: struct {
				Name string `json:"login"`
			}{Name: "test-owner"},
			CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			PushedAt:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	forkedRepos, err := fetchForkedRepos(
		context.Background(), // ctx
		mockServer.URL,       // baseURL
		"test-owner",         // owner
		"test-token",         // token
		10,                   // perPage
		1,                    // maxPage
	)
	if err != nil {
		t.Fatalf("fetchForkedRepos returned an error: %v", err)
	}

	if len(forkedRepos) != len(expected) {
		t.Fatalf("Expected %d forked repos, got %d", len(expected), len(forkedRepos))
	}

	for i, repo := range forkedRepos {
		if repo.Name != expected[i].Name ||
			repo.URL != expected[i].URL ||
			repo.IsFork != expected[i].IsFork ||
			repo.Owner.Name != expected[i].Owner.Name ||
			!repo.CreatedAt.Equal(expected[i].CreatedAt) ||
			!repo.UpdatedAt.Equal(expected[i].UpdatedAt) ||
			!repo.PushedAt.Equal(expected[i].PushedAt) {
			t.Errorf("Expected repo %+v, got %+v", expected[i], repo)
		}
	}
}

func TestDoRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		wantErr        bool
		errorContains  string
	}{
		{
			name:           "successful request",
			responseStatus: http.StatusOK,
			responseBody:   `{"success": true}`,
			wantErr:        false,
		},
		{
			name:           "API error response",
			responseStatus: http.StatusBadRequest,
			responseBody:   "Bad Request",
			wantErr:        true,
			errorContains:  "API request failed with status: 400",
		},
		{
			name:           "invalid JSON response",
			responseStatus: http.StatusOK,
			responseBody:   `{"success": true`, // Deliberately broken JSON
			wantErr:        true,
			errorContains:  "unexpected EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			server := httptest.NewServer(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(tt.responseStatus)
						fmt.Fprint(w, tt.responseBody) // Use Fprint to avoid newline
					}))
			defer server.Close()

			// Prepare request
			req, _ := http.NewRequest("GET", server.URL, nil)

			// Attempt to decode into this variable
			var result map[string]interface{}
			var token string

			// Call doRequest with the mock server's URL
			err := doRequest(req, token, &result)

			// Check for error existence
			if (err != nil) != tt.wantErr {
				t.Errorf("doRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check for specific error message, if applicable
			if tt.wantErr && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("doRequest() error = %v, want error to contain %v", err, tt.errorContains)
			}
		})
	}
}
func TestFilterForkedRepos_EmptyInput(t *testing.T) {
	t.Parallel()
	unguarded, guarded := filterForkedRepos(nil, nil, 30)
	if len(unguarded) != 0 || len(guarded) != 0 {
		t.Errorf("Expected both slices to be empty, got %v and %v", unguarded, guarded)
	}
}

func TestFilterForkedRepos_AllGuarded(t *testing.T) {
	now := time.Now()
	forkedRepos := []repo{
		{Name: "test-repo-1", CreatedAt: now, UpdatedAt: now, PushedAt: now},
		{Name: "test-repo-2", CreatedAt: now, UpdatedAt: now, PushedAt: now},
	}
	guardedRepoNames := []string{"test-repo"}
	unguarded, guarded := filterForkedRepos(forkedRepos, guardedRepoNames, 30)
	if len(unguarded) != 0 || len(guarded) != 2 {
		t.Errorf("Expected unguarded 0 and guarded 2, got unguarded %d and guarded %d", len(unguarded), len(guarded))
	}
}

func TestFilterForkedRepos_AllUnguardedDueToDate(t *testing.T) {
	forkedRepos := []repo{
		{
			Name:      "old-repo-1",
			CreatedAt: time.Now().AddDate(0, -1, 0),
			UpdatedAt: time.Now().AddDate(0, -1, 0),
			PushedAt:  time.Now().AddDate(0, -1, 0)},
		{
			Name:      "old-repo-2",
			CreatedAt: time.Now().AddDate(0, -2, 0),
			UpdatedAt: time.Now().AddDate(0, -2, 0),
			PushedAt:  time.Now().AddDate(0, -2, 0)},
	}
	var guardedRepoNames []string
	unguarded, guarded := filterForkedRepos(forkedRepos, guardedRepoNames, 10)

	if len(unguarded) != 2 || len(guarded) != 0 {
		t.Errorf("Expected unguarded 2 and guarded 0, got unguarded %d and guarded %d", len(unguarded), len(guarded))
	}
}

func TestFilterForkedRepos_UnknownGuardRepoName(t *testing.T) {
	forkedRepos := []repo{
		{
			Name:      "old-repo-1",
			CreatedAt: time.Now().AddDate(0, -1, 0),
			UpdatedAt: time.Now().AddDate(0, -1, 0),
			PushedAt:  time.Now().AddDate(0, -1, 0)},
		{
			Name:      "old-repo-2",
			CreatedAt: time.Now().AddDate(0, -2, 0),
			UpdatedAt: time.Now().AddDate(0, -2, 0),
			PushedAt:  time.Now().AddDate(0, -2, 0)},
	}
	guardedRepoNames := []string{"unknown-repo-1", "unknown-repo-2"}

	unguarded, guarded := filterForkedRepos(forkedRepos, guardedRepoNames, 10)

	if len(unguarded) != 2 || len(guarded) != 0 {
		t.Errorf("Expected unguarded 2 and guarded 0, got unguarded %d and guarded %d", len(unguarded), len(guarded))
	}
}

func TestFilterForkedRepos_MixedGuardedUnguarded(t *testing.T) {
	forkedRepos := []repo{
		{
			Name:      "new-repo-1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			PushedAt:  time.Now()},
		{
			Name:      "protected-old-repo",
			CreatedAt: time.Now().AddDate(0, -2, 0),
			UpdatedAt: time.Now().AddDate(0, -2, 0),
			PushedAt:  time.Now().AddDate(0, -2, 0)},
	}

	guardedRepoNames := []string{"protected"}
	unguarded, guarded := filterForkedRepos(forkedRepos, guardedRepoNames, 30)
	if len(unguarded) != 0 || len(guarded) != 2 {
		t.Errorf("Expected unguarded 0 and guarded 2, got unguarded %d and guarded %d", len(unguarded), len(guarded))
	}
}

func TestFilterForkedRepos_CaseInsensitive(t *testing.T) {
	forkedRepos := []repo{
		{
			Name:      "Case-Sensitive-Repo",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			PushedAt:  time.Now()},
	}
	guardedRepoNames := []string{"case-sensitive"}
	unguarded, guarded := filterForkedRepos(forkedRepos, guardedRepoNames, 30)
	if len(unguarded) != 0 || len(guarded) != 1 {
		t.Errorf("Expected unguarded 0 and guarded 1, got unguarded %d and guarded %d", len(unguarded), len(guarded))
	}
}

func TestFilterForkedRepos_MultipleMatches(t *testing.T) {
	forkedRepos := []repo{
		{
			Name:      "match-1",
			CreatedAt: time.Now().AddDate(0, -1, 0),
			UpdatedAt: time.Now().AddDate(0, -1, 0),
			PushedAt:  time.Now().AddDate(0, -1, 0)},
		{
			Name:      "match-2",
			CreatedAt: time.Now().AddDate(0, -2, 0),
			UpdatedAt: time.Now().AddDate(0, -2, 0),
			PushedAt:  time.Now().AddDate(0, -2, 0)},
	}
	guardedRepoNames := []string{"match-1", "match-2"}

	unguarded, guarded := filterForkedRepos(forkedRepos, guardedRepoNames, 29)
	if len(unguarded) != 0 || len(guarded) != 2 {
		t.Errorf("Expected unguarded 0 and guarded 2, got unguarded %d and guarded %d", len(unguarded), len(guarded))
	}
}

func TestDeleteRepo(t *testing.T) {
	t.Parallel()
	// Setup a local HTTP test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		// Respond with an empty JSON object to simulate a successful deletion
		fmt.Fprintln(w, "{}")
	}))
	defer server.Close()

	// Test the deleteRepo function
	ctx := context.Background()
	baseURL := server.URL // Use the test server URL
	owner := "testOwner"
	repoName := "testRepo"
	token := "testToken"

	err := deleteRepo(ctx, baseURL, owner, repoName, token)
	if err != nil {
		t.Errorf("deleteRepo() failed: %v", err)
	}
}

func TestDeleteRepos(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				// You can add more logic here to verify the request path, method, etc.
				fmt.Fprintln(w, "{}") // Simulate success
			}))
	defer server.Close()

	ctx := context.Background()
	baseURL := server.URL // Use the test server URL for the baseURL
	token := "testToken"
	repos := []repo{
		{Name: "testOwner/testRepo1", URL: ""},
		{Name: "testOwner/testRepo2", URL: ""},
	}

	err := deleteRepos(ctx, baseURL, token, repos)
	if err != nil {
		t.Errorf("deleteRepos() failed: %v", err)
	}
}

// Test cli flow

// Mock functions to replace actual behavior in tests
var (
	mockFlagErrorHandler = flag.ContinueOnError

	mockFetchForkedRepos = func(
		ctx context.Context,
		baseURL,
		owner,
		token string,
		perPage,
		maxPage int) ([]repo, error) {
		fmt.Println("mockFetchForkedRepos")
		return []repo{{Name: "test-repo"}}, nil
	}

	mockFilterForkedRepos = func(
		forkedRepos []repo,
		guardedRepoNames []string,
		olderThanDays int) ([]repo, []repo) {
		fmt.Println("mockFilterForkedRepos")
		return forkedRepos, nil
	}

	mockDeleteRepos = func(
		ctx context.Context,
		baseURL,
		token string,
		repos []repo) error {
		fmt.Println("mockDeleteRepos")
		return nil
	}
)

func TestNewCLIConfig_Defaults(t *testing.T) {
	t.Parallel()
	config := NewCLIConfig(nil, nil, "")

	if config.fetchForkedRepos == nil ||
		config.deleteRepos == nil ||
		config.flagErrorHandling != flag.ExitOnError {
		t.Fatal("Default functions were not set correctly")
	}
}

func TestWithFlagErrorHandling_Option(t *testing.T) {
	t.Parallel()
	config := NewCLIConfig(nil, nil, "").withFlagErrorHandling(mockFlagErrorHandler)
	if config.flagErrorHandling != mockFlagErrorHandler {
		t.Fatal("WithFlagErrorHandling did not set the flag error handling")
	}
}

func TestWithFetchForkedRepos_Option(t *testing.T) {
	t.Parallel()
	config := NewCLIConfig(nil, nil, "").withFetchForkedRepos(mockFetchForkedRepos)

	if config.fetchForkedRepos == nil {
		t.Fatal("WithFetchForkedRepos did not set the function")
	}
}

func TestWithFilterForkedRepos_Option(t *testing.T) {
	t.Parallel()
	config := NewCLIConfig(nil, nil, "").withFilterForkedRepos(filterForkedRepos)
	if config.filterForkedRepos == nil {
		t.Fatal("WithFilterForkedRepos did not set the function")
	}
}

func TestWithDeleteRepos_Option(t *testing.T) {
	t.Parallel()
	config := NewCLIConfig(nil, nil, "").withDeleteRepos(mockDeleteRepos)
	if config.deleteRepos == nil {
		t.Fatal("WithDeleteRepos did not set the function")
	}
}

func TestCLI_MissingOwnerToken(t *testing.T) {
	t.Parallel()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	cliConfig := NewCLIConfig(
		stdout,
		stderr,
		"test-version",
	).withFetchForkedRepos(mockFetchForkedRepos).
		withDeleteRepos(mockDeleteRepos).
		withFlagErrorHandling(mockFlagErrorHandler).
		withFilterForkedRepos(mockFilterForkedRepos)

		// Execute the CLI
	exitCode := cliConfig.CLI([]string{"cmd"})

	if !strings.Contains(stderr.String(), "owner and token are required") {
		t.Errorf("Expected error message not found in output")
	}

	if exitCode != 1 {
		t.Errorf("Expected os.Exit to be called once, got %d", exitCode)
	}
}
func TestCLI_Success(t *testing.T) {
	t.Parallel()

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	cliConfig := NewCLIConfig(
		stdout,
		stderr,
		"test-version",
	).withDeleteRepos(mockDeleteRepos).
		withFetchForkedRepos(mockFetchForkedRepos).
		withFlagErrorHandling(mockFlagErrorHandler).
		withFilterForkedRepos(mockFilterForkedRepos)

	// Execute the CLI
	args := []string{"--owner", "testOwner", "--token", "testToken", "--older-than-days", "30"}

	exitCode := cliConfig.CLI(args)

	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
	}
}
