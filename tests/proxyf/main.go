// HTTP-стенд возвращает маркер и параметры запроса для приёмки прокси.
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			http.Error(w, "read body failed", 400)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "overlay-marker\nhost=%s\nmethod=%s\nuri=%s\nbody=%s\n", r.Host, r.Method, r.RequestURI, body)
		log.Printf("HTTP request host=%s method=%s", r.Host, r.Method)
	})
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
