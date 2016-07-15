# miogo [![Build Status](https://travis-ci.org/efournival/miogo.svg?branch=master)](https://travis-ci.org/efournival/miogo)

## TODO
* Service arguments verification, better login check, admin check
* Create a default admin user at application startup if it does not exist
* Unit testing and direct HTTP service testing
* Travis CI (with Slack integration)
* File access with rights, refactor current user/group services
* Better logging (Logrus with Slack integration, for example)
* Documentation, GoDoc
* Admin (create and remove groups, define group admin, access everything)
* Operations (remove file, rename, copy, paste, lock, ...)
* Front-end and/or compatibility with Mioga API, WebDAV, syncing

## WebServices roadmap
* Logout
* SetResourceAccess(path[], access)
* MoveResource (path[], destPath, copy)
* DeleteResource (path[])

## Testing
http://dennissuratna.com/testing-in-go/

Use of the bundled Golang testing framework (`*_test.go` and `testing` package)

```
curl -v --data "email=test@test.test&password=test" http://localhost:8080/Login
```
```
curl -F "path=/test" -F "file=@/path/to/file" http://localhost:8080/Upload -b session=xxx
```
```
curl --data "path=/test" http://localhost:8080/GetFolder -b session=xxx
```
