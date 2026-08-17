# Go and chat

A simple chat service written in Go.

## Setup

Make sure you have Go 1.26.6+ installed to build the project.

Clone the repo:

```sh
git clone https://github.com/faergeek/go-and-chat.git
```

Then build the project:

```sh
go build
```

## Server

```sh
./go-and-chat server
```

Server listens on `localhost:1234` by default. A different address can be
provided as an argument:

```sh
./go-and-chat server localhost:4321 # different port on localhost
./go-and-chat server :3241          # listen on a given port on all interfaces
```

## Clients

```sh
./go-and-chat client
```

Client attempts to connect to `localhost:1234` by default. A different address
can be provided as an argument:

```sh
./client localhost:4321       # different port on localhost
./client 192.168.1.42:3241    # if server is on a different machine
```

You'll be first prompted to provide your name. Enter any name you like. If the
name is already taken, you'll have to come up with a different name.

After that you'll be presented with a prompt:

```
> |
```

Type a message and press Enter to send it.

Press `Ctrl + D` to quit.

Now Go and Chat!
