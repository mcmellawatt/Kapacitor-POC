# Kapacitor PoC with Fake DB

This is a simple proof of concept for using Kapacitor in Go with a simulated (fake) time-series database.

## Steps
1. Simulate time-series data in Go.
2. (Next) Integrate with Kapacitor client to define a task and send data.

## Usage

```sh
cd cmd
go run main.go
```

You should see simulated time-series data printed to the console.
