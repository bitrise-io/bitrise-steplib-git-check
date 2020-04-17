package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
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
	hostBaseURL  = "bitrise-steplib-git-check.herokuapp.com"
)

type githubtag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

type githubrelease struct {
	Body string `json:"body"`
}

type pullRequestModel struct {
	Action      string  `json:"action"`
	Number      int     `json:"number"`
	PullRequest content `json:"pull_request"`
}

type content struct {
	Merged bool   `json:"merged"`
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

func loadReleaseBody(giturl string, tag string) (string, error) {
	giturl = strings.TrimSuffix(giturl, ".git")
	giturl = strings.Replace(giturl, "https://github.com/", "https://api.github.com/repos/", 1)
	giturl = giturl + "/releases/tags/" + tag

	var release githubrelease

	if err := httpLoadJSON(giturl, &release); err != nil {
		return "", err
	}

	return release.Body, nil
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
		fmt.Println(url, string(b))
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
		if strings.HasSuffix(file.Filename, "/step.yml") && strings.HasPrefix(file.Filename, "steps/") {
			return true, nil
		}
	}

	return false, nil
}

func parseStep(prID string) (stepmanModels.StepModel, string, string, error) {
	var files []file
	if err := httpLoadJSON(fmt.Sprintf("https://api.github.com/repos/bitrise-io/bitrise-steplib/pulls/%s/files", prID), &files); err != nil {
		return stepmanModels.StepModel{}, "", "", err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Filename, "step.yml") && strings.HasPrefix(file.Filename, "steps/") {
			var yml stepmanModels.StepModel
			if err := httpLoadYML(file.RawURL, &yml); err != nil {
				return stepmanModels.StepModel{}, "", "", err
			}

			if yml.Source == nil {
				return stepmanModels.StepModel{}, "", "", fmt.Errorf("no source in step.yml")
			}

			versionDir := filepath.Dir(file.Filename)
			version := filepath.Base(versionDir)
			stepIDDir := filepath.Dir(versionDir)
			stepID := filepath.Base(stepIDDir)

			return yml, version, stepID, nil
		}
	}

	return stepmanModels.StepModel{}, "", "", fmt.Errorf("no step.yml found")
}

func createDiscourseTopic(title, body string) error {
	apiKey := os.Getenv("DISCOURSE_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("DISCOURSE_API_KEY is not set")
	}
	userName := os.Getenv("DISCOURSE_API_USERNAME")
	if userName == "" {
		return fmt.Errorf("DISCOURSE_API_USERNAME is not set")
	}
	category := os.Getenv("DISCOURSE_CATEGORY")
	if category == "" {
		return fmt.Errorf("DISCOURSE_CATEGORY is not set")
	}
	discourseURL := os.Getenv("DISCOURSE_URL")
	if discourseURL == "" {
		return fmt.Errorf("DISCOURSE_URL is not set")
	}

	formData := url.Values{}
	formData.Set("api_key", apiKey)
	formData.Set("api_username", userName)
	formData.Set("raw", body)
	formData.Set("category", category)
	formData.Set("title", title)

	resp, err := http.PostForm(discourseURL+"/posts.json", formData)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("Invalid response code: %d from: %s", resp.StatusCode, discourseURL+"/posts.json")
	}

	return nil
}
