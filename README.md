# Kapacitor PoC: Automated Invariant Checking with Go, InfluxDB, and Kapacitor

This proof of concept demonstrates how to use Go, InfluxDB, and Kapacitor to enforce time-series invariants (monotonicity and non-negativity) on simulated data streams. 

## What This PoC Does
- **Generates three test streams in Go:**
  - A valid stream (monotonic, non-negative)
  - A stream with a monotonicity violation
  - A stream with a non-negative violation
- **Injects all streams into InfluxDB** with a `stream` tag for partitioning.
- **Defines and uploads a Kapacitor batch task** (from `invariants.tick`) to check both invariants using TICKscript.
- **Starts a Go HTTP server** to receive Kapacitor alert POSTs.
- **Waits for both invariant violations to be detected** (alerts received), then exits automatically.
- **Automates setup/teardown** (Docker Compose, DB cleanup, etc) for rapid feedback and CI use.

## Usage

### With mise (recommended)

```sh
mise run run-poc
```

This will:
- Restart the Docker Compose stack
- Wait for Kapacitor to be ready
- Drop and recreate the InfluxDB database
- Run the Go PoC, inject data, and set up Kapacitor tasks
- Exit automatically when both invariant violations are detected