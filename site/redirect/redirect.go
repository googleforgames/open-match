package main

import (
	"net/http"

	"google.golang.org/appengine"
)

func main() {
	http.HandleFunc("/", redirect)
	appengine.Main()
}

func redirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://github.com/GoogleCloudPlatform/open-match/", 301)
}
