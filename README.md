# OVERVIEW
[![Go Reference](https://pkg.go.dev/badge/paepcke.de/npad.svg)](https://pkg.go.dev/paepcke.de/npad) [![Go Report Card](https://goreportcard.com/badge/paepcke.de/npad)](https://goreportcard.com/report/paepcke.de/npad) [![Go Build](https://github.com/paepckehh/npad/actions/workflows/golang.yml/badge.svg)](https://github.com/paepckehh/npad/actions/workflows/golang.yml)

[paepcke.de/npad](https://paepcke.de/npad/)

WebAPP to share, exchange and analyze secrets, keys, certificates and code in a secure way.

# KEYPOINTS

- Frontend: 100 % javascript free, small, fast
- Backend: 100 % pure go code, no cgo, no db, minimal external dependencies, secure storage 

## Transport 

- No legacy TLS downgrade support
- mutualTLS authentication (optional) removes large area of the golang application & tls stack attac surface

## Storage 

- No disk access at all, everything is compressed and encrypted in-memory (ram) 
- No accesslogs, no db, total stateless server
- No server side decryption key storage or knowledege at all 
- Any type fs storage is pure optional

## Executable 

- Chroots, drops privs, small resource footprint
- Minimlal startpage less than < 3kb, all UX elements embedded

## Configuration 

- BUILD TIME CONFIGURATION ONLY! Type safe configuration only!
- No unsafe runtime config files, commandline options or file parser!
- Details configuration: see server.go 
- Example configuration: see APP/npad/main.go 

## Anything else?

- Yes, its an quick hack, 
- No pre-build release binaries, build-time-configuration-only! 

# DOCS

[pkg.go.dev/paepcke.de/npad](https://pkg.go.dev/paepcke.de/npad)

# CONTRIBUTION

Yes, Please! PRs Welcome! 
