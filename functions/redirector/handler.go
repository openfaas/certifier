package function

import (
	"net/http"
	"os"
)

const defaultDestination = "https://google.com"

func Handle(w http.ResponseWriter, r *http.Request) {
	destination := os.Getenv("destination")
	if destination == "" {
		destination = defaultDestination
	}

	http.Redirect(w, r, destination, http.StatusFound)
}
