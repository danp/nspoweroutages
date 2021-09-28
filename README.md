# nspoweroutages

[Git scraping](https://simonwillison.net/2020/Oct/9/git-scraping/) of the data behind the [Nova Scotia Power Outage Map](http://outagemap.nspower.ca/).

## Why?

The outage map shows what's happening _right now_ but gives no way to see information about past outages, what has changed, etc. Regularly fetching the data that powers it and saving it in this way will let us see changes over time. That could let us answer questions like:

* What are the most commonly labeled causes of outages?
* What areas have frequent outages?
* How long do outages typically last?

And more!

## How it works

Every 10 minutes, a system of mine runs [bin/scrape.sh](bin/scrape.sh).

bin/scrape.sh:

1. fetches and combines outage data using [cmd/scrape-outages](cmd/scrape-outages), saving it to [data/outages.json](data/outages.json)
2. fetches the service area summary data, saving it to [data/report_servicearea.json](data/report_servicearea.json)

If there any changes from the current data, they're committed and pushed, like
[this](https://github.com/danp/nspoweroutages/commit/8183bda3b32f572e541caa6cd839b0d60b36bfba).

## Data format

`data/outages.json` has roughly this format:

```
[
  {
    "desc": {
      "cause": "<outage cause, such as Under Investigation, Damage to Equipment, High Winds>",
      "cluster": true|false, # whether this entry represents a number of outages or not
      "cust_a": { # customer affected info
        "masked": true|false, # when true, map UI would show "fewer than <val + 1> affected customers"
        "val": 5 # number of customers affected
      },
      "n_out": 3, # for cluster=true, how many outages this cluster covers
      "outages": [ # for cluster=true, individual outages in this cluster
        { "cause": "...", ... } # same structure as under `desc` here
      ],
      "etr": "2020-12-26T22:15:00-0400", # estimated time of recovery
      "start": "2020-12-26T14:07:00-0400" # start of outage
    },
    "geom": { # affected areas and/or points
      "a": [ # affected area(s)
        "en`nG~nbrKqp@qIoxBe{AoG}FaLs^nzDtpAbDr{@" # polyline encoding
      ],
      "p": [ # affected point(s)
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

cmd/scrape-outages starts by fetching [metadata.json](http://outagemap.nspower.ca/resources/data/external/interval_generation_data/metadata.json) which contains a single `directory` key pointing to the current data directory. At the time of this writing, it looks like this:

```json
{ "directory": "2020_12_27_19_18_00" }
```

This means the current data can be found under http://outagemap.nspower.ca/resources/data/external/interval_generation_data/2020_12_27_19_18_00/.

cmd/scrape-outages then fetches what I assume are the six data files for the map tiles visible when all of Nova Scotia is visible in the outage map, eg `2020_12_27_19_23_00/outages/030322.json` which contains data near HRM. Files are named based on their zoom level, the longer the filename the more zoomed in the data is.

These files have mostly the same format as the combined `outages.json` describe above.

cmd/scrape-outages zooms in to the areas covered by the initial six files by appending `0`, `1`, `2`, and `3` to the filename, then fetching that. For example, `030322.json` would zoom in to:

* `0303220.json`
* `0303221.json`
* `0303222.json`
* `0303223.json`

It continues to zoom in until either a deeper level returns a 404 (no data) or there only outages (and no clusters) in the returned data.

Finally, it combines all the data from the deepest possible levels into the single `outages.json`.
The resulting file may still have clusters in it if the deepest-found level contained them (example [here](https://github.com/danp/nspoweroutages/blob/d0cbdac3e322e22cc2b9f8b4ab213f10edff6e98/data/outages.json#L25-L30)).

(This is all based on observing what happens in the [Firefox Network Monitor](https://developer.mozilla.org/en-US/docs/Tools/Network_Monitor))
