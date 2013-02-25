# Tower

Tower makes your web development with Golang much more dynamic by monitoring your project file's changes and
re-run your app - yes, no more stopping and running manually! It will also show any compile error, panic
and runtime error through a clean html page (see the demo below).

[![Build Status](https://travis-ci.org/shaoshing/tower.png?branch=master)](https://travis-ci.org/shaoshing/tower)

## Demo

Watch at [Youtube](http://youtu.be/QRg7yWn1jzI)

## Install
```bash
go get github.com/shaoshing/tower
go install github.com/shaoshing/tower
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
localhost:8000
      \/
tower (listening to 8080)
      \/ (redirect)
your web app (listening to 5000)
```

When handling request of localhost:8000, tower is actually redirecting the request to your app by using Golang's _httputil.ReverseProxy_.
For the first request, tower will first compile and run your app in a child process. And for the subsequent requests, tower will rerun your app
if it find any change (using github.com/howeyc/fsnotify).

## License

Tower is released under the [MIT License](http://www.opensource.org/licenses/MIT).
