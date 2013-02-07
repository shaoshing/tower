# Tower

Recompile your web server if file changed automatically.

## Install
```bash
go install github.com/shaoshing/tower
```

## Usage


```bash
$ tower main.go
== Listening to http://localhost:8000


Started GET "/" at 2013-02-07 17:36:24 +700
== Building Server
== Starting Server
Completed in 1927ms



Started GET "/about" at 2013-02-07 17:36:31 +700
Completed in 0ms
2013/02/07 17:36:38 changed: test/server1.go



Started GET "/about" at 2013-02-07 17:36:39 +700
== Changed, stopping server
== Building Server
== Starting Server
Completed in 1942ms



Started GET "/about/home" at 2013-02-07 17:36:57 +700
Completed in 1ms
```
