package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gobuffalo/envy"
	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/tag", tagHandler).Methods("GET")
	router.HandleFunc("/update", updateHandler).Methods("POST")

	if err := http.ListenAndServe(":"+envy.Get("PORT", "8000"), router); err != nil {
		log.Fatal(err)
	}
}

func tagHandler(w http.ResponseWriter, r *http.Request) {
	setHeaders(w)

	prID := r.URL.Query().Get("pr")
	if prID == "" {
		if err := respondWithIcon(icnErr, w); err != nil {
			log.Fatal(err)
		}
		return
	}

	prDiff, err := getPRDiffLines(prID)
	if err != nil {
		log.Fatal(err)
		if err := respondWithIcon(icnErr, w); err != nil {
			log.Fatal(err)
		}
		return
	}

	version := filepath.Base(
		filepath.Dir(
			getLineContentAfterPrefix(prDiff, "+++ ")))
	giturl := getLineContentAfterPrefix(prDiff, "+  git: ")
	commit := getLineContentAfterPrefix(prDiff, "+  commit: ")

	if err := checkGithubTag(giturl, version, commit); err != nil {
		if err := respondWithIcon(icnErr, w); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := respondWithIcon(icnOk, w); err != nil {
		log.Fatal(err)
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Header.Get("X-Github-Event") {
	case "pull_request":
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
			return
		}
		var pr pullRequestModel
		if err := json.Unmarshal(b, &pr); err != nil {
			log.Fatal(err)
			return
		}

		switch pr.Action {
		case "opened":
			apiURL := fmt.Sprintf("https://api.github.com/repos/bitrise-io/bitrise-steplib/pulls/%d", pr.PullRequest.Number)
			newBody := map[string]interface{}{"body": fmt.Sprintf("![TagCheck](https://gogittag.herokuapp.com/tag?pr=%d)\r\n\r\n", pr.PullRequest.Number) + pr.PullRequest.Body}

			b, err := json.Marshal(newBody)
			if err != nil {
				log.Fatal(err)
				return
			}

			fmt.Println("apiurl:", apiURL, "new body:", string(b))

			c := http.Client{}
			req, err := http.NewRequest("PATCH", apiURL, bytes.NewReader(b))
			if err != nil {
				log.Fatal(err)
				return
			}

			req.Header.Add("Authorization", "token "+os.Getenv("GITHUB_USER")+":"+os.Getenv("GITHUB_ACCESS_TOKEN"))
			fmt.Println("Authorization " + "token " + os.Getenv("GITHUB_USER") + ":" + os.Getenv("GITHUB_ACCESS_TOKEN"))
			resp, err := c.Do(req)
			if err != nil {
				log.Fatal(err)
				return
			}

			bo, _ := ioutil.ReadAll(resp.Body)
			fmt.Println(string(bo))

			fmt.Printf("%+v\n", *resp)
			break
		}
		break
	}
}
