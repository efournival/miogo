# miogo

## TODO
1. Syncing, front-end interface, compatibility with the old API?
2. Security: AES?

## Roadmap
* Throw everything in the trash, except main entities (instances, groups, users)
* MongoDB
* Upload
* Browsing (list files, files info, ...)
* Entities (users, groups, instance ...)
* Login
* Access rights
* Operations (remove file, rename, copy, paste, lock, ...)
* Server infrastructure (Makefile, packages, ...)

## WebServices roadmap
* Upload (path, file[], unzip) => success or not
* GetFolder (path) => rights, files stat, creator info, ...
* GetFile (path) => binary file content
* NewFolder (path) => success or not
* MoveResource (path[], destPath, copy) => success or not
* DeleteResource (path[]) => success or not

## Testing
http://dennissuratna.com/testing-in-go/
Use of the bundled Golang testing framework (`*_test.go` and `testing` package)
