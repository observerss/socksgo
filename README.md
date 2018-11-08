# Socksgo

A minimal Socks5 Server that can switch outgoing IP


## Dependency

install [`dep`](https://golang.github.io/dep/docs/installation.html), then

```bash
dep ensure
```

## Building

```bash
bash build.sh
```

## Usage

```bash
NAME:
   Socksgo - a minimal socks5 server that can switch outgoing ip

USAGE:
   socksgo-darwin-amd64-0.1 [global options] command [command options] [arguments...]

VERSION:
   0.1

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --host value      host ip to listen on (default: "0.0.0.0")
   --port value      host port to listen on (default: "11080")
   --username value  socks5 username, optional
   --password value  socks5 password, optional
   --eip value       external ip to bind on (default: "0.0.0.0")
   --help, -h        show help
   --version, -v     print the version
```
