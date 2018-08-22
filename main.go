package main

import (
	"log"
	"net/http"

	"github.com/davecgh/go-spew/spew"

	"github.com/gobuffalo/envy"
	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/status", root).Methods("GET")

	log.Fatal(http.ListenAndServe(":"+envy.Get("PORT", "8000"), router))
}

func root(w http.ResponseWriter, r *http.Request) {
	spew.Dump(*r)
}
