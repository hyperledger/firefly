package apiserver

import (
	"net/http"
	"os"
	"path/filepath"
)

type UIHandler struct {
	staticPath string
	indexPath  string
	urlPrefix  string
}

func (h UIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path, err := filepath.Rel(h.urlPrefix, r.URL.Path)
	if err != nil {
		// if we failed to get the path respond with a 400 bad request
		// and stop
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// prepend the path with the path to the static directory
	path = filepath.Join(h.staticPath, path)

	// check whether a file exists at the given path
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		// file does not exist, serve index.html
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	} else if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// otherwise, use http.FileServer to serve the static dir
	http.StripPrefix(h.urlPrefix, http.FileServer(http.Dir(h.staticPath))).ServeHTTP(w, r)
}
