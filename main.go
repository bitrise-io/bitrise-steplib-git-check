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
	// setting image mime type and no-cache
	setHeaders(w)

	prID := r.URL.Query().Get("pr")
	if prID == "" {
		if err := respondWithIcon(icnErr, w); err != nil {
			fmt.Println(err)
		}
		return
	}

	yml, version, err := parseStepYML(prID)
	if err != nil {
		if err := respondWithIcon(icnErr, w); err != nil {
			fmt.Println(err)
		}
		return
	}

	versionParts := strings.Split(version, ".")

	if len(versionParts) != 3 {
		if err := respondWithIcon(icnErrSemver, w); err != nil {
			fmt.Println(err)
		}
		return
	}

	for _, part := range versionParts {
		if _, err := strconv.Atoi(part); err != nil {
			if err := respondWithIcon(icnErrSemver, w); err != nil {
				fmt.Println(err)
			}
			return
		}
	}

	if err := checkGithubTag(yml.Source.Git, version, yml.Source.Commit); err != nil {
		if err := respondWithIcon(icnErrCommit, w); err != nil {
			fmt.Println(err)
		}
		return
	}

	if err := respondWithIcon(icnOk, w); err != nil {
		fmt.Println(err)
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get("X-Github-Event") != "pull_request" {
		return
	}

	var pr pullRequestModel
	if err := json.NewDecoder(r.Body).Decode(&pr); err != nil {
		fmt.Println(err)
		return
	}

	if pr.Action == "opened" {
		exists, err := isPRHasStepYML(fmt.Sprintf("%d", pr.Number))
		if err != nil {
			fmt.Println(err)
			return
		}

		if !exists {
			return
		}

		if strings.Contains(pr.PullRequest.Body, fmt.Sprintf("https://%s/tag?pr=%d", hostBaseURL, pr.Number)) {
			return
		}

		stepDefinition, version, err := parseStepYML(fmt.Sprintf("%d", pr.Number))
		if err != nil {
			fmt.Println(err)
			return
		}

		apiURL := fmt.Sprintf("https://api.github.com/repos/bitrise-io/bitrise-steplib/pulls/%d", pr.Number)
		badgeContent := fmt.Sprintf("![TagCheck](https://%s/tag?pr=%d)\r\n\r\n", hostBaseURL, pr.Number)
		releaseURLContent := ""

		if strings.Contains(stepDefinition.Source.Git, "/bitrise-io/") || strings.Contains(stepDefinition.Source.Git, "/bitrise-steplib/") || strings.Contains(stepDefinition.Source.Git, "/bitrise-community/") {
			releaseURLContent = fmt.Sprintf("%s/releases/%s\r\n\r\n", strings.TrimSuffix(stepDefinition.Source.Git, ".git"), version)
		}

		newBody := map[string]interface{}{
			"body": badgeContent + releaseURLContent + pr.PullRequest.Body,
		}

		b, err := json.Marshal(newBody)
		if err != nil {
			fmt.Println(err)
			return
		}

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
	}

	if pr.Action == "closed" && pr.PullRequest.Merged {
		stepDefinition, version, err := parseStepYML(fmt.Sprintf("%d", pr.Number))
		if err != nil {
			fmt.Println(err)
			return
		}

		if strings.Contains(stepDefinition.Source.Git, "/bitrise-io/") || strings.Contains(stepDefinition.Source.Git, "/bitrise-steplib/") || strings.Contains(stepDefinition.Source.Git, "/bitrise-community/") {
			if stepDefinition.Title == nil {
				return
			}

			title := *stepDefinition.Title + " v" + version
			body, err := loadReleaseBody(stepDefinition.Source.Git, version)
			if err != nil {
				fmt.Println(err)
				return
			}

			// append git release url
			body += "\n\n\n" + fmt.Sprintf("%s/releases/%s\r\n\r\n", strings.TrimSuffix(stepDefinition.Source.Git, ".git"), version)

			if err := createDiscourseTopic(title, body); err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}
