# OVERVIEW

[paepche.de/npad](https://paepcke.de/npad/)

WebAPP to share, exchange and analyze secrets, keys, certificates and code in a secure way.

# KEYPOINTS

- Frontend: 100 % javascript free, secure transport
- Backend: 100 % pure go code, no cgo, no db, minimal external dependencies, secure storage 

## Transport 

- No legacy TLS downgrade support
- Optional mutualTLS authentication removes large area of the golang application & tls stack attac surface

## Storage 

- No disk access at all, everything is compressed and encrypted in-memory (ram) 
- No accesslogs, no db, total stateless server
- No server side decryption key knowledge (encoded within client access url)
- Any type fs storage is pure optional

## Executable 

- Chroots, drops privs, small resource footprint
- Minimlal startpage less than < 3kb, all UX elements embedded

## Configuration 

- BUILD TIME CONFIGURATION! Type safe configuration only!
- No unsafe runtime config files, commandline options or file parser!
- Details configuration: see server.go 
- Example configuration: see APP/npad/main.go 

## Anything else?

- Yes, its an quick hack
- No pre-build binaries, its build-time-configuration! 

# DOCS

[pkg.go.dev/paepcke.de/npad](https://pkg.go.dev/paepcke.de/npad)

# CONTRIBUTION

Yes, Please! PRs Welcome! 
