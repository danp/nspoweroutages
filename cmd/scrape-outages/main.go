package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

// Fetch the JSON data used by http://outagemap.nspower.ca/external/default.html and
// combine into a single file.

func main() {
	var f fetcher
	data, err := f.fetch()
	if err != nil {
		log.Fatal(err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.Encode(data)
}

const defaultBaseURL = "http://outagemap.nspower.ca/resources/data/external/interval_generation_data"

type fetcher struct {
	baseURL string
}

func (f *fetcher) fetch() ([]json.RawMessage, error) {
	curl, err := f.currentURL()
	if err != nil {
		return nil, err
	}

	data := make(map[string][]json.RawMessage)

	// These are the top level tile (?) ids fetched when the map is in the full view.
	const basePathLen = 6 // len of initial ids below
	ids := []string{"030231", "030233", "030320", "030321", "030322", "030323"}
	for len(ids) > 0 {
		id := ids[0]
		ids = ids[1:]

		of, err := f.fetchOutageFile(curl, "outages/"+id)
		if err != nil {
			return nil, err
		}

		if len(of.FileData) == 0 {
			continue
		}

		// Deeper data is preferred so delete anything captured a level above.
		if len(id) > basePathLen {
			delete(data, id[:len(id)-1])
		}

		anycluster := false

		for _, fd := range of.FileData {
			if fd.Desc.Cluster {
				anycluster = true
			}
			data[id] = append(data[id], fd.Body)
		}

		if anycluster {
			for i := 0; i < 4; i++ {
				ids = append(ids, id+strconv.Itoa(i))
			}
		}

	}

	// Explicitly making an empty slice so it will
	// marshal as [].
	out := make([]json.RawMessage, 0)
	for _, fds := range data {
		out = append(out, fds...)
	}
	return out, nil
}

func (f *fetcher) fetchOutageFile(curl, path string) (outageFile, error) {
	var of outageFile

	fd, err := fetch(curl + "/" + path + ".json")
	if err != nil {
		return of, nil
	}

	if len(fd) == 0 {
		return of, nil
	}

	var rof rawOutageFile
	if err := json.Unmarshal(fd, &rof); err != nil {
		return of, err
	}

	of.FileData = make([]outageFileData, 0, len(rof.FileData))
	for _, rfd := range rof.FileData {
		var fd outageFileData
		if err := json.Unmarshal(rfd, &fd); err != nil {
			return of, err
		}
		fd.Body = rfd
		of.FileData = append(of.FileData, fd)
	}

	return of, nil
}

func (f *fetcher) currentURL() (string, error) {
	baseURL := f.baseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	mdb, err := fetch(baseURL + "/metadata.json")
	if err != nil {
		return "", fmt.Errorf("fetching medatata: %w", err)
	}

	var r struct {
		Directory string
	}
	if err := json.Unmarshal(mdb, &r); err != nil {
		return "", fmt.Errorf("decoding metadata: %w", err)
	}

	return baseURL + "/" + r.Directory, nil
}

// fetches URL u, returns error from http.Get or
// if status not 200 or 404. On 404, returned data
// will be nil.
func fetch(u string) ([]byte, error) {
	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad status %d", resp.StatusCode)
	}

	return b, nil
}

type outageFileData struct {
	Desc struct {
		Cluster bool
	}

	Body json.RawMessage `json:"-"`
}

type outageFile struct {
	FileData []outageFileData `json:"file_data"`
}

type rawOutageFile struct {
	FileData []json.RawMessage `json:"file_data"`
}
