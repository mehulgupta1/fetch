package cli

// Version of fetch.
const Version = "0.1.0"

// Banner is printed to stderr on help / bare invocation.
const Banner = `
   ███████╗███████╗████████╗ ██████╗██╗  ██╗
   ██╔════╝██╔════╝╚══██╔══╝██╔════╝██║  ██║
   █████╗  █████╗     ██║   ██║     ███████║
   ██╔══╝  ██╔══╝     ██║   ██║     ██╔══██║
   ██║     ███████╗   ██║   ╚██████╗██║  ██║
   ╚═╝     ╚══════╝   ╚═╝    ╚═════╝╚═╝  ╚═╝
   js file collector · 6 sources · one clean list      v` + Version + `
   ─────────────────────────────────────────────────────────`

// Help is the usage text printed under the banner.
const Help = Banner + `

 INPUT / OUTPUT
   -l <file|->        url list. feeds grep only. "-" or piped stdin = read stdin
   -d <file|domain>   target(s): a file of domains OR one domain.
                      feeds subjs + getJS + katana + hakrawler + urlscan
   -o <file>          output file            (default: js.txt)
   --silent           no banner/report; urls to stdout (for automation)
   --debug            verbose: full tool stderr + internal decisions
   -timeout <dur>     per-tool timeout        (default: 10m)

 CRAWL CONTROL  (katana + hakrawler)
   --depth <n>        crawl depth            (default: 2)
   --subs             include subdomains in crawl scope
   --rate <n>         max requests/sec       (katana only)
   -c <n>             concurrency / threads  (default: 10)
   --headless         katana browser mode - catches JS-rendered links (slow)

 REQUEST CONTROL
   --proxy <url>      route through a proxy (e.g. Burp)
   -H "K: V"          add request header (repeatable) - authed crawling
   -k, --insecure     skip TLS certificate verification

 URLSCAN (source S6)
   --urlscan-key <k>  urlscan api key (else URLSCAN_API_KEY env, else config)
   --urlscan-limit <n> newest recordings to open (default 20 / 100 with key)

 CLEANING
   --in-scope         drop JS outside the -d target scope (needs -d)

 COMMANDS
   -setup             install the 4 tools (subjs, getJS, katana, hakrawler)
   -config            store the urlscan api key
   -h, --help         show this help
   --version          show version (fetch + detected tool versions)

 EXAMPLES
   fetch -l urls.txt -o js.txt
   fetch -d subs.txt
   cat urls.txt | fetch -d example.com --silent | nuclei
   fetch -setup
`
