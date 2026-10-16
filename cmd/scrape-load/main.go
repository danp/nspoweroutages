package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	up, err := do()
	if err != nil {
		log.Fatal(err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.Encode(up)
}

type point struct {
	Key string
	Val float64
}

type update struct {
	Time   time.Time
	Points []point
}

func do() (update, error) {
	resp, err := http.Get("https://resourcesprd-nspower.aws.silvertech.net/oasis/current_report.shtml")
	if err != nil {
		return update{}, fmt.Errorf("get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return update{}, fmt.Errorf("get: unexpected status %s", resp.Status)
	}

	return parseReport(resp.Body)
}

func parseReport(r io.Reader) (update, error) {
	loc, err := time.LoadLocation("America/Halifax")
	if err != nil {
		return update{}, fmt.Errorf("load location: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return update{}, fmt.Errorf("new document: %w", err)
	}

	var up update

	// The current conditions timestamp is outside the table. Later sections
	// have their own timestamps and tables, which must not be included.
	ts := strings.TrimSpace(doc.Find(".updated-header span").First().Text())
	ts = strings.TrimSpace(strings.TrimPrefix(ts, "Last Updated:"))
	if ts == "" {
		return update{}, fmt.Errorf("no time found")
	}
	t, err := time.ParseInLocation("02-Jan-06 15:04", ts, loc)
	if err != nil {
		return update{}, fmt.Errorf("bad time %q: %w", ts, err)
	}
	up.Time = t.UTC()

	var findErr error
	doc.Find("table.two-col").First().Find("tbody tr").Each(func(_ int, sel *goquery.Selection) {
		tds := sel.Find("td")
		if tds.Length() != 2 {
			return
		}

		key := strings.TrimSpace(tds.First().Text())
		val := strings.TrimSpace(tds.Next().Text())

		fv, err := strconv.ParseFloat(val, 64)
		if err != nil {
			findErr = fmt.Errorf("bad val %q: %w", val, err)
			return
		}

		up.Points = append(up.Points, point{Key: key, Val: fv})
	})
	if findErr != nil {
		return update{}, fmt.Errorf("scraping: %w", findErr)
	}

	if len(up.Points) == 0 {
		return update{}, fmt.Errorf("no current system conditions found")
	}

	sort.Slice(up.Points, func(i, j int) bool {
		return up.Points[i].Key < up.Points[j].Key
	})

	return up, nil
}
