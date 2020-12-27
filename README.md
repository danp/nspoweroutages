# nspoweroutages

[Git scraping](https://simonwillison.net/2020/Oct/9/git-scraping/) of the data behind the [Nova Scotia Power Outage Map](http://outagemap.nspower.ca/).

## Why?

The outage map shows what's happening _right now_ but gives no way to see information about past outages, what has changed, etc. Regularly fetching the data that powers it and saving it in this way will let us see changes over time. That could let us answer questions like:

* What are the most commonly labeled causes of outages?
* What areas have frequent outages?
* How long do outages typically last?

And more!

## How it works

Every [10 minutes or so](.github/workflows/scheduled.yml#L9), GitHub Actions runs [bin/scrape](bin/scrape).

bin/scrape:

1. fetches and combines outage data using [cmd/scrape](cmd/scrape), saving it to [data/outages.json](data/outages.json)
2. fetches the service area summary data, saving it to [data/report_servicearea.json](data/report_servicearea.json)
3. commits and pushes any changes, like [this](https://github.com/danp/nspoweroutages/commit/c274f7c18f4c797aabede5c8a7fbdcfa24dcf136)

## Data format

`data/outages.json` has roughly this format:

```
[
  {
    "desc": {
      "cause": "<outage cause, such as Under Investigation, Damage to Equipment, High Winds>",
      "cluster": true|false, # whether this entry represents a number of outages or not
      "cust_a": { # customer affected info
        "val": 5 # number of customers affected
      },
      "n_out": 3, # for cluster=true, how many outages this cluster covers
      "etr": "2020-12-26T22:15:00-0400", # estimated time of recovery,
      "start": "2020-12-26T14:07:00-0400" # start of outage
    },
    "geom": { # affected area, "p" for point(s) or "a" for area(s)
      "a": [
        "en`nG~nbrKqp@qIoxBe{AoG}FaLs^nzDtpAbDr{@" # polyline encoding
      ],
      "p": [ # specific point(s)
        "gowuGjud~J" # polyline encoding
      ]
    },
    "id": "5",
    "title": "<Outage Information or Area Outage>"
  },
  ...
]
```

Unfortunately, `id` is not maintained across updates. This means when going from 5 outages to 4, ids will likely be shuffled, making it difficult to determine which outage was removed.

`geom.a` and `geom.p` are in [polyline encoding](https://developers.google.com/maps/documentation/utilities/polylinealgorithm) and can be decoded with eg [polyline.DecodeCoords](https://pkg.go.dev/github.com/twpayne/go-polyline#DecodeCoords) or interactively using the [Interactive Polyline Encoder Utility](https://developers.google.com/maps/documentation/utilities/polylineutility).

## How data fetching works

cmd/scrape starts by fetching [metadata.json](http://outagemap.nspower.ca/resources/data/external/interval_generation_data/metadata.json) which contains a single `directory` key pointing to the current data directory. At the time of this writing, it looks like this:

```json
{ "directory": "2020_12_27_19_18_00" }
```

This means the current data can be found under http://outagemap.nspower.ca/resources/data/external/interval_generation_data/2020_12_27_19_18_00/.

cmd/scrape then fetches what I assume are the six data files for the map tiles visible when all of Nova Scotia is visible in the outage map, eg `2020_12_27_19_23_00/outages/030322.json` which contains data near HRM. Files are named based on their zoom level, the longer the filename the more zoomed in the data is.

These files have mostly the same format as the combined `outages.json` describe above.

cmd/scrape zooms in to the areas covered by the initial six files by appending `0`, `1`, `2`, and `3` to the filename, then fetching that. For example, `030322.json` would zoom in to:

* `0303220.json`
* `0303221.json`
* `0303222.json`
* `0303223.json`

It continues to zoom in until either a deeper level returns a 404 (no data) or there only outages (and no clusters) in the returned data.

Finally, it combines all the data from the deepest possible levels into the single `outages.json`.

(This is all based on observing what happens in the [Firefox Network Monitor](https://developer.mozilla.org/en-US/docs/Tools/Network_Monitor))
