<div align="center">
<pre align="center">
<h1 align="center">
fork
sweeper
|
\|/
-|-
// \\
/// \\\
//// \\\\
</h1>
<h4 align="center">
Remove unused GitHub forks
</h4>
</pre>
</div>

## Installation

-   On macOS, brew install:

    ```sh
    brew tap rednafi/fork-sweeper https://github.com/rednafi/fork-sweeper \
        && brew install fork-sweeper
    ```

-   Elsewhere, go install:

    ```sh
    go install fork-sweeper
    ```

## Prerequisites

-   Collect your GitHub API [access token]. The token will be sent as `Bearer <token>` in
    the HTTP header while making API requests.
-   The token must have write and delete access to the forked repos.
-   Set the `GITHUB_TOKEN` to your current shell environment with
    `export GITHUB_TOKEN=<token>` command.

## Usage

-   Run help:

    ```sh
    fork-sweeper -h
    ```

    ```txt
    Usage of fork-sweeper:
      -delete
            Delete forked repos
      -max-page int
            Maximum page number to fetch (default 100)
      -older-than-days int
            Delete forked repos older than this number of days (default 60)
      -owner string
            GitHub repo owner (required)
      -per-page int
            Number of forked repos fetched per page (default 100)
      -token string
            GitHub access token (required)
      -version
    ```

-   List forked repos older than `n` days. By default it'll fetch all reposties that were
    forked at least 60 days ago.

    ```sh
    fork-sweeper --owner rednafi --token $GITHUB_TOKEN --older-than-days 60
    ```

-   The CLI won't delete any repository unless you explicitly tell it to do so with the
    `--delete` flag:

    ```sh
    fork-sweeper --owner rednafi --token $GITHUB_TOKEN --delete
    ```

[access token]:
    https://docs.github.com/en/rest/authentication/authenticating-to-the-rest-api?apiVersion=2022-11-28
