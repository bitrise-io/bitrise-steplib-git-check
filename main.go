package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/gobuffalo/envy"
	"github.com/gorilla/mux"
)

const (
	icnErr = "cross.svg"
	icnOk  = "ok.svg"
)

type gittag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/tag", tag).Methods("GET")

	if err := http.ListenAndServe(":"+envy.Get("PORT", "8000"), router); err != nil {
		log.Fatal(err)
	}
}

func tag(w http.ResponseWriter, r *http.Request) {
	setHeaders(w)

	prURL := r.URL.Query().Get("pr")

	if prURL == "" {
		if err := respondWithIcon(icnErr, w); err != nil {
			log.Fatal(err)
		}
		return
	}

	prDiff, err := getPRDiff(prURL)
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

	var tags []gittag

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

func getPRDiff(url string) ([]string, error) {
	r, err := http.Get(url + ".diff")
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
