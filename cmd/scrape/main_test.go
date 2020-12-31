package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFetch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metadata.json", func(w http.ResponseWriter, r *http.Request) {
		var resp struct {
			Directory string `json:"directory"`
		}
		resp.Directory = "current-directory"
		if err := json.NewEncoder(w).Encode(&resp); err != nil {
			t.Error(err)
		}
	})

	type respFileData struct {
		Desc struct {
			Cluster bool `json:"cluster"`
		} `json:"desc"`
		ID        string `json:"id"`
		OtherData string `json:"other_data"`
	}
	type resp struct {
		FileData []respFileData `json:"file_data"`
	}

	mux.HandleFunc("/current-directory/outages/", func(w http.ResponseWriter, r *http.Request) {
		fn := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]

		var fds []respFileData
		switch fn {
		case "030231.json":
			// Only an outage at the top level for this path so it's returned directly.
			fds = []respFileData{
				{
					ID:        fn + "-A",
					OtherData: "Hello",
				},
			}
		case "030233.json", "0302332.json":
			// Descending down this path returns a cluster for the first two levels.
			fds = []respFileData{
				{
					ID: fn + "-C-AB",
				},
				{
					ID: fn + "-C",
				},
			}
			fds[0].Desc.Cluster = true
		case "03023321.json":
			// The cluster is expanded at the third level so these are the entries
			// that should be returned.
			fds = []respFileData{
				{
					ID: fn + "-A",
				},
				{
					ID: fn + "-B",
				},
				{
					ID: fn + "-C",
				},
			}
		default:
			return
		}

		var re resp
		re.FileData = fds
		if err := json.NewEncoder(w).Encode(&re); err != nil {
			t.Error(err)
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	var f fetcher
	f.baseURL = srv.URL

	data, err := f.fetch()
	if err != nil {
		t.Fatal(err)
	}

	var recv []respFileData
	for _, d := range data {
		var fd respFileData
		if err := json.Unmarshal(d, &fd); err != nil {
			t.Fatal(err)
		}
		recv = append(recv, fd)
	}

	want := []respFileData{
		{
			ID:        "030231.json-A",
			OtherData: "Hello",
		},
		{
			ID: "03023321.json-A",
		},
		{
			ID: "03023321.json-B",
		},
		{
			ID: "03023321.json-C",
		},
	}

	sorter := func(s []respFileData) func(i, j int) bool {
		return func(i, j int) bool {
			return s[i].ID < s[j].ID
		}
	}
	sort.Slice(recv, sorter(recv))
	sort.Slice(want, sorter(want))

	if d := cmp.Diff(want, recv); d != "" {
		t.Errorf("fetch mismatch (-want +got):\n%s", d)
	}
}

func TestFetchNoOutages(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/metadata.json", func(w http.ResponseWriter, r *http.Request) {
		var resp struct {
			Directory string `json:"directory"`
		}
		resp.Directory = "current-directory"
		if err := json.NewEncoder(w).Encode(&resp); err != nil {
			t.Error(err)
		}
	})
	// All data files 404.

	srv := httptest.NewServer(mux)
	defer srv.Close()

	var f fetcher
	f.baseURL = srv.URL

	data, err := f.fetch()
	if err != nil {
		t.Fatal(err)
	}

	if len(data) != 0 {
		t.Errorf("wanted empty data")
	}

	if data == nil {
		t.Errorf("wanted non-nil data")
	}
}
