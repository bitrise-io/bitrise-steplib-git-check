package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gobuffalo/envy"
	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	router := mux.NewRouter()

	////
	// handlers
	//

	router.HandleFunc("/tag", tagHandler).Methods("GET")
	router.HandleFunc("/update", updateHandler).Methods("POST")

	//
	////

	if err := http.ListenAndServe(":"+envy.Get("PORT", "8000"), router); err != nil {
		fmt.Println(err)
	}
}

func tagHandler(w http.ResponseWriter, r *http.Request) {
	////
	// initializing handler
	//

	// setting image mime type and no-cache
	setHeaders(w)

	// parsing pr query from url
	prID := r.URL.Query().Get("pr")
	if prID == "" {
		if err := respondWithIcon(icnErr, w); err != nil {
			fmt.Println(err)
		}
		return
	}

	//
	////

	// get the step.yml and the required infos from the pr
	version, giturl, commit, err := parseStepYML(prID)
	if err != nil {
		if err := respondWithIcon(icnErr, w); err != nil {
			fmt.Println(err)
		}
		return
	}

	////
	// checking semver format
	//

	versionParts := strings.Split(version, ".")

	// check if icon has 3 parts
	if len(versionParts) != 3 {
		if err := respondWithIcon(icnErrSemver, w); err != nil {
			fmt.Println(err)
		}
		return
	}

	// check if all 3 parts is castable to int
	for _, part := range versionParts {
		if _, err := strconv.Atoi(part); err != nil {
			if err := respondWithIcon(icnErrSemver, w); err != nil {
				fmt.Println(err)
			}
			return
		}
	}

	//
	////

	////
	// check git tag in the author's repo if it is not moved from the commit hash in the step.yml
	//

	if err := checkGithubTag(giturl, version, commit); err != nil {
		if err := respondWithIcon(icnErrCommit, w); err != nil {
			fmt.Println(err)
		}
		return
	}

	//
	///

	////
	// everything ok
	//

	if err := respondWithIcon(icnOk, w); err != nil {
		fmt.Println(err)
	}

	//
	///
}

// webhook wired in the github repo
func updateHandler(w http.ResponseWriter, r *http.Request) {
	////
	// handle only pull request
	//
	if r.Header.Get("X-Github-Event") != "pull_request" {
		return
	}

	var pr pullRequestModel
	if err := yaml.NewDecoder(r.Body).Decode(&pr); err != nil {
		fmt.Println(err)
		return
	}

	if pr.Action != "opened" {
		return
	}

	//
	////

	////
	// check if the pr just opened has step.yml in it
	//
	fmt.Println(pr)

	exists, err := isPRHasStepYML(fmt.Sprintf("%d", pr.PullRequest.Number))
	if err != nil {
		fmt.Println(err)
		return
	}

	if !exists {
		return
	}

	//
	////
	if strings.Contains(pr.PullRequest.Body, fmt.Sprintf("https://gogittag.herokuapp.com/tag?pr=%d", pr.PullRequest.Number)) {
		return
	}

	////
	// updating the PR's initial comment section: append badge as first element
	//

	apiURL := fmt.Sprintf("https://api.github.com/repos/bitrise-io/bitrise-steplib/pulls/%d", pr.PullRequest.Number)
	badgeContent := fmt.Sprintf("![TagCheck](https://gogittag.herokuapp.com/tag?pr=%d)\r\n\r\n", pr.PullRequest.Number)
	newBody := map[string]interface{}{
		"body": badgeContent + pr.PullRequest.Body,
	}

	// convert new body message to json
	b, err := json.Marshal(newBody)
	if err != nil {
		fmt.Println(err)
		return
	}

	// call authenticated PATCH request
	c := http.Client{}
	req, err := http.NewRequest("PATCH", apiURL, bytes.NewReader(b))
	if err != nil {
		fmt.Println(err)
		return
	}

	req.SetBasicAuth(os.Getenv("GITHUB_USER"), os.Getenv("GITHUB_ACCESS_TOKEN"))
	_, err = c.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	//
	////
}
