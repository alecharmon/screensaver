# screensaver

A terminal screensaver CLI with an animated starfield background and a centered Quote of the Day panel.

## Install

```bash
go install ./cmd/screensaver
```

## Build

```bash
go build -o screensaver ./cmd/screensaver
```

This creates a local binary at:

`./screensaver`

Make sure your Go bin path is on `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Configure

Set your ZenQuotes API key (optional for some usage tiers, recommended):

```bash
export ZENQUOTES_API_KEY="your_key_here"
```

## Run

```bash
screensaver
```

Controls:

- `n` fetches the next quote and updates today's cache
- any other keypress or mouse input exits the app

## Cache behavior

The quote response is cached and refreshed at most once per local calendar day.
If you press `n`, the newly fetched quote is saved as today's cached quote and
will persist until the day changes.

Cache file:

`$(os.UserCacheDir)/screensaver/quote_of_day.json`
