package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
)

const (
	icnErr = "assets/cross.svg"
	icnOk  = "assets/ok.svg"
)

type githubtag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

// type githubpr struct {
// 	URL     string `json:"url"`
// 	DiffURL string `json:"diff_url"`
// 	State   string `json:"state"`
// 	Body    string `json:"body"`
// }

type pullRequestModel struct {
	Action      string  `json:"action"`
	PullRequest content `json:"pull_request"`
}

type content struct {
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

func checkGithubTag(giturl string, tag string, commit string) error {
	giturl = strings.TrimSuffix(giturl, ".git")
	giturl = strings.Replace(giturl, "https://github.com/", "https://api.github.com/repos/", 1)
	giturl = giturl + "/tags"

	r, err := http.Get(giturl)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	var tags []githubtag

	if err := json.Unmarshal(b, &tags); err != nil {
		return err
	}

	for _, t := range tags {
		if t.Name == tag && t.Commit.SHA == commit {
			return nil
		}
	}

	return fmt.Errorf("not found")
}

func getLinesAfterLineHasPrefix(s []string, prefix string) []string {
	started := false
	for _, line := range s {
		if !started {
			if strings.HasPrefix(line, prefix) {
				started = true
			}
			s = s[1:]
			continue
		}
	}
	return s
}

func getLineContentAfterPrefix(s []string, prefix string) (f string) {
	for _, line := range s {
		if strings.HasPrefix(line, prefix) {
			f = line
			break
		}
	}
	f = strings.TrimPrefix(f, prefix)
	return
}

func trimLinesPrefixes(lns []string, s string) []string {
	for i, l := range lns {
		lns[i] = strings.TrimPrefix(l, s)
	}
	return lns
}

func getPRDiffLines(id string) ([]string, error) {
	r, err := http.Get(fmt.Sprintf("https://github.com/bitrise-io/bitrise-steplib/pull/%s", id) + ".diff")
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(b), "\n"), nil
}

func setHeaders(w http.ResponseWriter) {
	w.Header().Add("Content-Type", "image/svg+xml")
	w.Header().Add("Cache-Control", "no-cache")
}

func respondWithIcon(icn string, w http.ResponseWriter) error {
	b, err := fileutil.ReadBytesFromFile(icn)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
