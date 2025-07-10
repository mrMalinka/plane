package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	const port = 3000
	const dir = "./src/app/src/main/assets/"

	portS := fmt.Sprintf(":%d", port)
	http.Handle("/", http.FileServer(http.Dir(dir)))
	log.Printf("Serving on http://localhost%s\n", portS)
	log.Fatal(http.ListenAndServe(portS, nil))
}
