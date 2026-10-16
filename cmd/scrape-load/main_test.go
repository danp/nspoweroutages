package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestParseReport(t *testing.T) {
	f, err := os.Open("testdata/current_report.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	got, err := parseReport(f)
	if err != nil {
		t.Fatal(err)
	}
	want := update{
		Time: time.Date(2026, time.October, 16, 22, 57, 0, 0, time.UTC),
		Points: []point{
			{Key: "Cape Breton Export", Val: 229.1},
			{Key: "East End Export (at East Bay)", Val: 269.1},
			{Key: "East End Export (at Sydney)", Val: 67.2},
			{Key: "Flow Into Metro", Val: 303.5},
			{Key: "Highlands Export", Val: -5.2},
			{Key: "Maritime Link Import", Val: 154.7},
			{Key: "NS Export", Val: -67.5},
			{Key: "Net Load", Val: 1354.4},
			{Key: "Onslow Import", Val: 334.2},
			{Key: "Onslow South", Val: 455.1},
			{Key: "Valley Import", Val: 103.1},
			{Key: "Western Import", Val: 101.1},
			{Key: "Wind Generation", Val: 533.9},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("report mismatch (-want +got):\n%s", diff)
	}
}

func TestParseReportErrors(t *testing.T) {
	const header = `<div class="updated-header"><span>Last Updated: <i>16-Oct-26 19:57</i></span><span>MW</span></div>`
	for _, tt := range []struct {
		name string
		html string
		want string
	}{
		{"missing timestamp", `<html></html>`, "no time found"},
		{"invalid timestamp", strings.Replace(header, "16-Oct-26", "invalid", 1), "bad time"},
		{"missing conditions", header, "no current system conditions found"},
		{"invalid value", header + `<table class="two-col"><tr><td>Net Load</td><td>unavailable</td></tr></table>`, "bad val"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseReport(strings.NewReader(tt.html))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got error %v, want %q", err, tt.want)
			}
		})
	}
}
