# Directive

`dnstap /tmp/dnstap.sock true`

* Socket path
* Report the wire-format DNS message.

# dnstap command line tool

```sh
go get github.com/dnstap/golang-dnstap
cd $GOPATH/src/github.com/dnstap/golang-dnstap/dnstap
go build
./dnstap -u /tmp/dnstap.sock
./dnstap -u /tmp/dnstap.sock -y
```

There is a buffer, expect at least 13 requests before the server sends its dnstap messages to the socket.
