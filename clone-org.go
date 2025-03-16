// Package cloneorg contains useful functions to find and clone a github
// organization repositories.
package cloneorg

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/go-github/v51/github"
	"golang.org/x/oauth2"
)

// Repo represents the repository data we need.
type Repo struct {
	Name string
	URL  string
}

// ErrClone happens when a git clone fails.
var ErrClone = errors.New("git clone failed")

// ErrCreateDir happens when we fail to create the target directory.
var ErrCreateDir = errors.New("failed to create directory")

var sem = make(chan bool, 20)

// Clone a given repository into a given destination.
func Clone(repo Repo, destination string) error {
	sem <- true
	defer func() {
		<-sem
	}()

	// nolint: gosec
	cmd := exec.Command(
		"git", "clone", "--depth", "1", repo.URL,
		filepath.Join(destination, repo.Name),
	)
	if bts, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %v: %v", ErrClone, repo.Name, string(bts))
	}
	return nil
}

// AllOrgRepos finds all repositories of a given organization.
func AllOrgRepos(token, org string) (repos []Repo, err error) {
	ctx := context.Background()
	client := github.NewClient(oauth2.NewClient(
		ctx,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}),
	))
	result, err := findRepos(ctx, client, org)
	if err != nil {
		log.Printf("Unexpected error in findRepos: %v", err)
		return nil, err // Only return error if findRepos itself fails critically
	}
	for _, repo := range result {
		repos = append(repos, Repo{
			Name: *repo.Name,
			URL:  *repo.SSHURL,
		})
	}
	if len(repos) == 0 {
		log.Printf("Warning: No repositories fetched for %s. Check token permissions or organization name.", org)
	}
	return repos, nil
}

const pageSize = 30

func findRepos(ctx context.Context, client *github.Client, org string) (result []*github.Repository, err error) {
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: pageSize},
	}
	for {
		log.Printf("Fetching repositories for %s, page %d...", org, opt.ListOptions.Page)
		repos, resp, err := client.Repositories.ListByOrg(ctx, org, opt)
		if err != nil {
			log.Printf("Error fetching repositories for %s on page %d: %v. Skipping this page.", org, opt.ListOptions.Page, err)
			// Skip this page and move to the next one if possible
			if resp != nil && resp.NextPage != 0 {
				opt.ListOptions.Page = resp.NextPage
				continue
			}
			// If no next page, break and return what we have
			break
		}
		result = append(result, repos...)
		log.Printf("Fetched %d repositories on page %d", len(repos), opt.ListOptions.Page)
		if resp.NextPage == 0 {
			log.Printf("No more pages to fetch for %s", org)
			break
		}
		opt.ListOptions.Page = resp.NextPage
	}
	// Return the repositories fetched so far, even if some pages failed
	return result, nil
}

// CreateDir creates the directory if it does not exists.
func CreateDir(directory string) error {
	stat, err := os.Stat(directory)
	directoryDoesNotExists := err != nil

	if directoryDoesNotExists {
		err := os.MkdirAll(directory, 0o700)
		if err != nil {
			return fmt.Errorf("%w: %s: %s", ErrCreateDir, directory, err.Error())
		}

		return nil
	}

	if stat.IsDir() {
		return nil
	}

	return fmt.Errorf("%w: %s is a file", ErrCreateDir, directory)
}
