# smee-client-go

A Golang-based client for smee.io, a service that delivers webhooks to your local development environment.
It can work over coorporate proxy


## Usage

### CLI

The `smee-client` command will forward webhooks from smee.io to your local development environment. It also supports github's authenticated header.

Run `smee-client --help` for usage.

```
$ ./smee-client --help
NAME:
   smee-client-go - smee-client that support proxy

USAGE:
   smee-client-go [global options] command [command options] [arguments...]

VERSION:
   develop

COMMANDS:
   start    Start smee-client
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug        Display debug output (default: false)
   --no-color     No print color (default: false)
   --help, -h     show help (default: false)
   --version, -v  print the version (default: false)
```


```
./smee-client start help

NAME:
   smee-client-go start - Start smee-client

USAGE:
   smee-client-go start [command options] [arguments...]

OPTIONS:
   --url value      URL of the webhook proxy service. Required. For exemple: https://smee.io/VyOocXe0HCKwlSj)
   --target value   Full URL (including protocol and path) of the target service the events will forwarded to. Required. For exemple: http://jenkins.mycompany.local:8080/github-webhook/
   --secret value   Secret to be used for HMAC-SHA1 secure hash calculation
   --timeout value  The timeout to wait when access on URL and target (default: 2m0s)
   --help, -h       show help (default: false
```
