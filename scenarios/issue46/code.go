package code

import (
	"net/http"
)

func RunMe() (string) {
    resp, _ := http.Get("http://www.google.com")
    return resp.Status
}
