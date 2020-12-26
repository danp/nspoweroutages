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
	ids := []string{"030231", "030233", "030320", "030321", "030322", "030323"}
	var paths [][]string
	for _, id := range ids {
		paths = append(paths, []string{id})
	}
	for len(paths) > 0 {
		p := paths[0]
		paths = paths[1:]
		id := p[len(p)-1]

		of, err := f.fetchOutageFile(curl, "outages/"+id)
		if err != nil {
			return nil, err
		}

		if len(of.FileData) == 0 {
			continue
		}

		// Deeper data is preferred so delete anything captured a level above.
		if len(p) > 1 {
			delete(data, p[len(p)-2])
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
				newp := make([]string, len(p)+1)
				copy(newp, p)
				newp[len(newp)-1] = id + strconv.Itoa(i)
				paths = append(paths, newp)
			}
		}

	}

	var out []json.RawMessage
	for _, fds := range data {
		for _, fd := range fds {
			out = append(out, fd)
		}
	}

	return out, nil
}

func (f *fetcher) fetchOutageFile(curl, path string) (outageFile, error) {
	var of outageFile

	fd, err := f.fetchFile(curl, path)
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

func (f *fetcher) fetchFile(curl, path string) ([]byte, error) {
	resp, err := http.Get(curl + "/" + path + ".json")
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad status %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (f *fetcher) currentURL() (string, error) {
	baseURL := f.baseURL
	if baseURL == "" {
		baseURL = "http://outagemap.nspower.ca/resources/data/external/interval_generation_data"
	}

	resp, err := http.Get(baseURL + "/metadata.json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("bad status %d", resp.StatusCode)
	}

	var r struct {
		Directory string
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}

	return baseURL + "/" + r.Directory, nil
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
