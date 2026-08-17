<h1 align="center">fetch</h1>

<p align="center">
  <b>A JS file collector for bug hunters — 6 sources, one clean list.</b>
</p>

<p align="center">
  <a href="#installation">Install</a> •
  <a href="#setup">Setup</a> •
  <a href="#usage">Usage</a> •
  <a href="#flags">Flags</a> •
  <a href="#how-it-works">How it works</a>
</p>

---

```
   ███████╗███████╗████████╗ ██████╗██╗  ██╗
   ██╔════╝██╔════╝╚══██╔══╝██╔════╝██║  ██║
   █████╗  █████╗     ██║   ██║     ███████║
   ██╔══╝  ██╔══╝     ██║   ██║     ██╔══██║
   ██║     ███████╗   ██║   ╚██████╗██║  ██║
   ╚═╝     ╚══════╝   ╚═╝    ╚═════╝╚═╝  ╚═╝
   js file collector · 6 sources · one clean list
```

**fetch** hunts down every JavaScript file for a target by running the best
JS-finding tools **in parallel**, then merging their output into one clean,
deduplicated list. It combines an existing URL list, live crawlers, and a
passive cloud source — so you catch JS that any single tool would miss.

## Features

- **6 sources in one command** — internal `.js` grep + `subjs` + `getJS` (from a URL list) and `katana` + `hakrawler` + `urlscan.io` (from live domains).
- **Live progress** — see each source start/finish and a heartbeat while long crawls run.
- **Clean output** — merged, exact-deduplicated, sorted `.js` list.
- **Scope control** — `--in-scope` drops out-of-scope JS (CDNs, third-party).
- **Pipeline-friendly** — reads URLs from stdin, `--silent` for clean stdout.
- **Self-installing** — `fetch -setup` installs the 4 external tools for you.

## Installation

Install the tool globally (just like `subfinder`/`katana`) — this drops the
`fetch` binary into `$(go env GOPATH)/bin` (usually `~/go/bin`):

```sh
go install github.com/mehulgupta1/fetch@latest
```

Make sure `~/go/bin` is on your `PATH`:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

> Requires **Go 1.25+**. Linux and macOS only.

## Setup

`fetch` orchestrates four external tools. Install them all with one command:

```sh
fetch -setup
```

This installs `subjs`, `getJS`, `katana`, and `hakrawler` (all via `go install`).

### urlscan.io API key (recommended)

The `urlscan` source needs a free API key to read scan details (anonymous
access is rate-limited/blocked). Grab one from
[urlscan.io/user/profile](https://urlscan.io/user/profile) and save it:

```sh
fetch -config              # interactive prompt
# or
fetch -config --urlscan-key <YOUR_KEY>
```

The key is stored in `~/.config/fetch/config` (chmod `600`). `fetch` also reads
`--urlscan-key` and the `URLSCAN_API_KEY` env var.

## Usage

```sh
# I already have a URL list — just pull the .js out of it
fetch -l urls.txt

# Crawl + urlscan a set of live domains
fetch -d subs.txt

# A single domain
fetch -d example.com

# Everything: list source + crawl + urlscan, keep only in-scope JS
fetch -l urls.txt -d subs.txt --in-scope -o js.txt

# Pipeline: read from stdin, clean stdout for the next tool
cat urls.txt | fetch -d example.com --silent | nuclei
```

By default the clean list is written to `js.txt`. A run finishes with a
per-source report:

```
-------- fetch done (21s) --------
 sources
   subjs        9
   getJS        9
   katana       2
   hakrawler    0
   urlscan    377
 found   397 js  ->  209 unique
 saved   js.txt
----------------------------------
```

## Flags

```
INPUT / OUTPUT
  -l <file|->         url list (feeds grep). "-" or piped stdin = read stdin
  -d <file|domain>    target(s): a file of domains OR one domain
  -o <file>           output file (default: js.txt)
  --silent            no banner/report; urls to stdout (automation)
  --debug             verbose: full tool stderr + internal decisions
  -timeout <dur>      per-tool timeout (default: 10m)

CRAWL CONTROL  (katana + hakrawler)
  --depth <n>         crawl depth (default: 2)
  --exact             restrict to the exact host (default: crawl subdomains)
  --rate <n>          max requests/sec, katana (default: 100)
  -c <n>              concurrency / threads (default: 5)
  --headless          katana browser mode (catches JS-rendered links)

REQUEST CONTROL
  --proxy <url>       route through a proxy (e.g. Burp)
  -H "K: V"           add a request header (repeatable)
  -k, --insecure      skip TLS certificate verification

URLSCAN
  --urlscan-key <k>   urlscan api key (else env / config)
  --urlscan-limit <n> newest recordings to open (default 20 / 100 with key)

CLEANING
  --in-scope          drop JS outside the -d target scope

COMMANDS
  -setup              install the 4 tools
  -config             store the urlscan api key
  -h, --help          show help
  --version           show version + detected tool versions
```

## How it works

```
-l urls.txt  ──►  grep (.js)                              (you already have urls)
-d domains   ──►  subjs · getJS · katana · hakrawler · urlscan
                        │
                        ▼
              filter to .js  →  --in-scope  →  dedup + sort  →  js.txt
```

- **`-l`** feeds only the internal `.js` grep — instant, no requests.
- **`-d`** feeds the two page-scrapers (`subjs`, `getJS`), two crawlers
  (`katana`, `hakrawler`), and the passive cloud source (`urlscan.io`).
- Every source's output is filtered to real `.js` URLs, merged, deduplicated
  (loose — cache-buster query strings are kept), sorted, and written out.

## Notes

- Runs all sources in parallel for speed. On a fragile target you can slow
  things with `--rate` and lower `-c`.
- `fetch` is a security-recon tool — only use it against targets you are
  authorized to test.

## License

[MIT](LICENSE)
