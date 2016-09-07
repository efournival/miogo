# Miogo [![Build Status](https://travis-ci.org/efournival/miogo.svg?branch=master)](https://travis-ci.org/efournival/miogo)

Miogo is the codename for a refreshed and improved version of Mioga2, a collaborative and secure extranet software. For now, it will only implement the features of the file browser, Magellan.

Miogo is meant to be better in every ways and focused on security and performance.

Here are the enhancements compared to Mioga2:
* Security: using Bcrypt and only Bcrypt for user authentication, easy integration of end-to-end AES-based encryption is planned.
* Performance: preliminary benchmarks are showing an improvement up to 300 times faster regarding files operations. It completely outperforms Mioga2 when it comes to memory use (up to 100x more efficient) and CPU load.
* Scalability: Miogo is designed to scale very well by design with the use of MongoDB and the Go language. It also relies extensively on caching and stores most of its data in RAM.
* Maintenance: less code doing more things, faster.
* Ease of use: clone, compile, launch; setup Miogo in two minutes.
* Reliability under heavy operations: no more crashes caused by lack of RAM and swapping due to the way Apache2 handles a lot of connections.

## What still has to be done
* Perfect handling of files rights
* Implementation of some key features of Magellan (file versioning, comments)
* Documentation with GoDoc
* WebDAV
* Syncing application


## Wanna test?
At the moment, Miogo is only a back-end and cannot be used out of the box.

However there is a complete test suite which can be run with `go test -v`.

The Miogo executable program can be compiled with `go build` and run with `./miogo`. Don't forget to create a new configuration file named `miogo.conf`.

Miogo can be benchmarked by passing [POST data](https://github.com/wg/wrk/issues/22) to [WRK](https://github.com/wg/wrk).

## Direct testing
```
curl -c cookies.txt --data "email=test@test.test&password=test" http://localhost:8080/Login
```
```
curl -b cookies.txt -F "path=/test" -F "file=@/path/to/file" http://localhost:8080/Upload
```
```
curl -b cookies.txt --data "path=/test" http://localhost:8080/GetFolder -b session=xxx
```
