package src

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestUnmarshalRepo tests the unmarshalling of a JSON string into a repo struct
func TestUnmarshalRepo(t *testing.T) {
	t.Parallel()
	// Example JSON string that represents a repo's data
	jsonString := `{
		"full_name": "example/repo",
		"html_url": "https://github.com/example/repo",
		"fork": false,
		"owner": {
			"name": "example"
		},
		"created_at": "2020-01-01T00:00:00Z"
	}`

	// Expected repo object based on the JSON string
	expected := repo{
		Name:   "example/repo",
		URL:    "https://github.com/example/repo",
		IsFork: false,
		Owner: struct {
			Name string `json:"name"`
		}{
			Name: "example",
		},
		CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Unmarshal the JSON string into a repo struct
	var result repo
	err := json.Unmarshal([]byte(jsonString), &result)
	if err != nil {
		t.Fatalf("Unmarshalling failed: %v", err)
	}

	// Compare the expected and actual repo structs
	if !reflect.DeepEqual(expected, result) {
		t.Errorf("Unmarshalled repo does not match expected value. Expected %+v, got %+v", expected, result)
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
				`[{"full_name": "example/forkedrepo", "html_url": "https://github.com/example/forkedrepo", "fork": true, "owner": {"name": "example"}, "created_at": "2020-01-01T00:00:00Z"}]`)
		}))
	defer mockServer.Close()

	expected := []repo{
		{
			Name:   "example/forkedrepo",
			URL:    "https://github.com/example/forkedrepo",
			IsFork: true,
			Owner: struct {
				Name string `json:"name"`
			}{Name: "example"},
			CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	forkedRepos, err := fetchForkedReposPage(
		context.Background(), mockServer.URL, "example", "fake-token", 1, 10)
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
			!repo.CreatedAt.Equal(expected[i].CreatedAt) {
			t.Errorf("Expected repo %+v, got %+v", expected[i], repo)
		}
	}
}

func TestFetchForkedRepos(t *testing.T) {
	t.Parallel()
	mockServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `[{"full_name": "example/forkedrepo",`+
				`"html_url": "https://test.com/example/forkedrepo", "fork": true,`+
				`"owner": {"name": "example"}, "created_at": "2020-01-01T00:00:00Z"},`+
				`{"full_name": "example/forkedrepo2",`+
				`"html_url": "https://test.com/example/forkedrepo2", "fork": true,`+
				`"owner": {"name": "example2"}, "created_at": "2020-01-01T00:00:00Z"}]`)

		}))

	defer mockServer.Close()

	expected := []repo{
		{
			Name:   "example/forkedrepo",
			URL:    "https://test.com/example/forkedrepo",
			IsFork: true,
			Owner: struct {
				Name string `json:"name"`
			}{Name: "example"},
			CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			Name:   "example/forkedrepo2",
			URL:    "https://test.com/example/forkedrepo2",
			IsFork: true,
			Owner: struct {
				Name string `json:"name"`
			}{Name: "example2"},
			CreatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	forkedRepos, err := fetchForkedRepos(
		context.Background(), mockServer.URL, "example", "fake-token", 10, 1)
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
			!repo.CreatedAt.Equal(expected[i].CreatedAt) {
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

			// Call doRequest with the mock server's URL
			err := doRequest(req, &result)

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

func TestPrintWithColor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		color string
		text  string
		want  string
	}{
		{"\033[31m", "Hello, Red!", "\033[31mHello, Red!\033[0m\n"},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {

			// Redirect stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Capture output
			output := make(chan string)
			go func() {
				var buf bytes.Buffer
				io.Copy(&buf, r)
				output <- buf.String()
			}()

			// Execute function
			printWithColor(tt.color, tt.text)

			// Restore stdout
			w.Close()
			os.Stdout = old

			// Read output
			got := <-output

			// Verify output
			if got != tt.want {
				t.Errorf("printWithColor() got = %v, want %v", got, tt.want)
			}
		})
	}
}
