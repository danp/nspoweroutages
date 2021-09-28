package main

import (
	"bytes"
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
	loc, err := time.LoadLocation("America/Halifax")
	if err != nil {
		return update{}, fmt.Errorf("load location: %w", err)
	}

	resp, err := http.Get("https://resourcesprd-nspower.aws.silvertech.net/oasis/current_report.shtml")
	if err != nil {
		return update{}, fmt.Errorf("get: %w", err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return update{}, fmt.Errorf("get: %w", err)
	}

	// html body table.standard tbody tr td

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(b))
	if err != nil {
		return update{}, fmt.Errorf("new document: %w", err)
	}

	var up update

	var findErr error
	var done bool
	doc.Find("html body table.standard tbody tr").Each(func(_ int, sel *goquery.Selection) {
		if done {
			return
		}

		tds := sel.Find("td")
		if strings.Contains(tds.Text(), "Five Day Historic") {
			done = true
			return
		}
		if tds.Length() != 2 {
			return
		}

		key := strings.TrimSpace(tds.First().Text())
		val := strings.TrimSpace(tds.Next().Text())

		// Last Updated: 28-Sep-21 19:32:05
		if strings.HasPrefix(key, "Last Updated: ") {
			ts := strings.TrimPrefix(key, "Last Updated: ")
			t, err := time.ParseInLocation("02-Jan-06 15:04:05", ts, loc)
			if err != nil {
				findErr = fmt.Errorf("bad time %q: %w", ts, err)
				return
			}
			up.Time = t.UTC()
			return
		}

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

	if up.Time.IsZero() {
		return update{}, fmt.Errorf("no time found")
	}

	sort.Slice(up.Points, func(i, j int) bool {
		return up.Points[i].Key < up.Points[j].Key
	})

	return up, nil
}
