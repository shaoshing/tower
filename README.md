# Tower

Tower makes your Go web development much more dynamic by monitoring file's changes in your project and then re-run your 
app to apply those changes – yeah, no more stopping and running manually! It will also show compiler error, panic and 
runtime error through a clean page (see the demo below).

[![Build Status](https://travis-ci.org/shaoshing/tower.png?branch=master)](https://travis-ci.org/shaoshing/tower)

## Demo

Watch at [Youtube](http://youtu.be/QRg7yWn1jzI)

## Install
```bash
go get github.com/shaoshing/tower
```

## Usage

```bash
cd your/project
tower # now visit localhost:8000
```

Tower will, by default, assume your web app's main file is _main.go_ and the port is _5000_. These can be changed by:

```bash
tower -m app.go -p 3000
```

Or put them in a config file:

```bash
tower init
vim .tower.yml
tower
```

## Troubleshooting

#### 'Too many open files'

Run the following command to increase the number of files that a process can open:

```bash
ulimit -S -n 2048 # tested on OSX
```

## How it works?

```
browser: http://localhost:8000
      \/
tower (listening 8000)
      \/ (reverse proxy)
your web app (listening 5000)
```

Any request comes from localhost:8000 will be handled by Tower and then be redirected to your app. The redirection is 
done by using _[httputil.ReverseProxy](http://golang.org/pkg/net/http/httputil/#ReverseProxy)_. Before redirecting the request, Tower will compile and run your app in 
another process if your app heaven't been run or there is file been changed; Tower is using 
_[howeyc/fsnotify](https://github.com/howeyc/fsnotify)_ to monitor file changes.

## License

Tower is released under the [MIT License](http://www.opensource.org/licenses/MIT).
