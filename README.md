# miogo

## TODO
* Service arguments verification
* Entities: users and groups
* Login
* Access control with a custom architecture, i.e. find a clean way of passing current user rights and groups when calling a service, implement a fast method for rights checking, handle errors; https://github.com/efournival/openpt/blob/master/server.go
* Admin (create and remove groups, define group admin, access everything)
* Operations (remove file, rename, copy, paste, lock, ...)
* Front-end and/or compatibility with Mioga API, WebDAV, syncing

## WebServices roadmap
* Login => success or not
* Logout => success
* AddUserToGroup (user, group) => success or not
* RemoveUserFromgroup (user, group) => success or not
* CreateGroup (group, admin) => success or not
* RemoveGroup (group) => success or not
* SetGroupAdmin (group, admin) => success or not
* SetResourceAccess(path[], access) => success or not
* MoveResource (path[], destPath, copy) => success or not
* DeleteResource (path[]) => success or not

## Testing
http://dennissuratna.com/testing-in-go/

Use of the bundled Golang testing framework (`*_test.go` and `testing` package)

```
curl --data "path=/test" http://localhost:8080/NewFolder
```
```
curl -F "path=/test" -F "file=@/path/to/file" http://localhost:8080/Upload
```
```
curl --data "path=/test" http://localhost:8080/GetFolder
```
```
curl --data "path=/test/file" http://localhost:8080/GetFile >file
```
