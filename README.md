## HTTP Flight Recorder

In Go 1.25, the standard library trace introduced the the flight recorder which was explained by this blog post.

This module provides an HTTP wrapper for the library to trigger snapshots based on remote requests.

**Note**: _Use of this module is insecure by default. SSL certificate validation should be used._ 

## A flight recorder service which provides an HTTP interface to interacting with apps.

API endpoints the service might expose.

```
POST /recorder/start
POST /recorder/stop
POST /recorder/update
GET  /recorder/status
GET  /recorder/snapshot
```

## Requirements

Exposing application state from the endpoint is insecure by default.

My recommendation is that SSL certificates will need to be registered to the server.

## GET  /recorder/status

Gets the status of the flight recorder:

* Enabled: bool
* SetPeriod: Duration
* SetSize: bytes

## POST /recorder/start

Starts the flight recorder if it is stopped.

## POST /recorder/stop

Stops the flight recorder if it is running.

## GET  /recorder/snapshot

Provides the snapshot of the flight recorder.

Returns HTTP errors when existing snapshot request is being processed, or flight recorder is stopped.

500 internal service error.

## POST /recorder/update

Update SetPeriod and SetSize of flight recorder.

### Later roadmap:

* TLS / SSL cert configuration.
Metrics for flight recorders:
* Number of requests.
* Successes, failures, panics, timeouts, etc.


