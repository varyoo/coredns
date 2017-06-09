# Directive

`dnstap /tmp/db true`: directive name, dnstap binary database path, report the wire-format dns message (true/false).

# dnstap command line tool

```sh
go get github.com/dnstap/golang-dnstap
cd $GOPATH/src/github.com/dnstap/golang-dnstap/dnstap
go build
./dnstap -r /tmp/db
./dnstap -r /tmp/db -y
```

# dndstap binary database VS socket

The database can become hudge with time, and cannot be cleared while the server is running that
easly. This is why I believe the socket is prefered.

However the dnstap golang library https://github.com/dnstap/golang-dnstap dosen't support yet bidirectional framestreams.
Which are used when reading a socket by the dnstap tool, so before it can be used in the middleware, I will have to work on adding a bidirectional encoder into the golang library...
https://github.com/farsightsec/golang-framestream/issues/1
