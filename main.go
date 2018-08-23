package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

	prURL := r.URL.Query().Get("pr")
	if prURL == "" {
		if err := respondWithIcon(icnErr, w); err != nil {
			log.Fatal(err)
		}
		return
	}

	prDiff, err := getPRDiffLines(prURL)
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

func updatePRs() error {
	// r, err := http.Get("https://api.github.com/repos/bitrise-io/bitrise-steplib/pulls?state=open")
	// if err != nil {
	// 	return err
	// }

	// b, err := ioutil.ReadAll(r.Body)
	// if err != nil {
	// 	return err
	// }

	// //var prs []githubpr
	// if err := json.Unmarshal(b, &prs); err != nil {
	// 	return err
	// }

	// for _, pr := range prs {
	// 	if !strings.Contains(pr.Body, "![TagCheck](https://gogittag.herokuapp.com/tag?pr=") {
	// 		// update here
	// 	}
	// }

	return nil
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Header.Get("X-Github-Event") {
	case "pull_request":
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			// return err
		}

		var pr pullRequestModel

		if err := json.Unmarshal(b, &pr); err != nil {
			// return err
		}

		fmt.Printf("%#v\n", pr)
		break
	}
}
