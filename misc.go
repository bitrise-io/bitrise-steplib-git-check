package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/go-utils/fileutil"

	stepmanModels "github.com/bitrise-io/stepman/models"
)

const (
	icnOk        = "assets/ok.svg"
	icnErr       = "assets/cross.svg"
	icnErrSemver = "assets/invalid-semver.svg"
	icnErrCommit = "assets/invalid-commit.svg"
)

type githubtag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

type pullRequestModel struct {
	Action      string  `json:"action"`
	PullRequest content `json:"pull_request"`
}

type content struct {
	Number int    `json:"number"`
	Body   string `json:"body"`
}

type file struct {
	Filename string `json:"filename"`
	RawURL   string `json:"raw_url"`
}

func checkGithubTag(giturl string, tag string, commit string) error {
	giturl = strings.TrimSuffix(giturl, ".git")
	giturl = strings.Replace(giturl, "https://github.com/", "https://api.github.com/repos/", 1)
	giturl = giturl + "/tags"

	var tags []githubtag

	if err := httpLoadJSON(giturl, &tags); err != nil {
		return err
	}

	for _, t := range tags {
		if t.Name == tag && t.Commit.SHA == commit {
			return nil
		}
	}

	return fmt.Errorf("not found")
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

func httpLoadJSON(url string, model interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(b, model); err != nil {
		fmt.Println(string(b))
		return err
	}
	return nil
}

func httpLoadYML(url string, model interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	if err := yaml.NewDecoder(r.Body).Decode(model); err != nil {
		return err
	}
	return nil
}

func isPRHasStepYML(prID string) (bool, error) {
	var files []file
	if err := httpLoadJSON(fmt.Sprintf("https://api.github.com/repos/bitrise-io/bitrise-steplib/pulls/%s/files", prID), &files); err != nil {
		return false, err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Filename, "step.yml") {
			return true, nil
		}
	}

	return false, nil
}

func parseStepYML(prID string) (version string, url string, commit string, err error) {
	var files []file
	if err := httpLoadJSON(fmt.Sprintf("https://api.github.com/repos/bitrise-io/bitrise-steplib/pulls/%s/files", prID), &files); err != nil {
		return "", "", "", err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Filename, "step.yml") {
			var yml stepmanModels.StepModel
			if err := httpLoadYML(file.RawURL, &yml); err != nil {
				return "", "", "", err
			}

			version = filepath.Base(filepath.Dir(file.Filename))
			url = yml.Source.Git
			commit = yml.Source.Commit

			return version, url, commit, nil
		}
	}

	return "", "", "", nil
}
