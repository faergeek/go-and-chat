# Go and chat

Chat server and client, implemented in Go.

## Motivation

I was always wondering what goes into building terminal user interfaces, so I
decided to learn more about that topic and built this chat app to apply what I
have learned.

## Quick Start

- Make sure you have Go 1.26.6+ installed to build the project.
- Clone the repo:
  ```sh
  git clone https://github.com/faergeek/go-and-chat.git
  ```
- Run the server:
  ```sh
  go run . server
  ```
- In another terminal run the first client:
  ```sh
  go run . client
  ```
  Provide a name, e.g. "John Doe".
- In yet another terminal run the second client:
  ```sh
  go run . client
  ```
  Provide a different name, e.g. "Jane Doe"
- Now go and chat!
- To quit, press `Ctrl + C`.

## Usage

Examples below assume you've built the binary, here's how you do it:

```sh
go build
```

### Server

```sh
./go-and-chat server
```

Server listens on `localhost:1234` by default. A different address can be
provided as an argument:

```sh
./go-and-chat server localhost:4321 # different port on localhost
./go-and-chat server 0.0.0.0:3241   # listen on a given port on all interfaces
```

### Client

```sh
./go-and-chat client
```

Client attempts to connect to `localhost:1234` by default. A different address
can be provided as an argument:

```sh
./go-and-chat client localhost:4321       # different port on localhost
./go-and-chat client 192.168.1.42:3241    # if server is on a different machine
```

You'll be first prompted to provide your name. Enter any name you like. If the
name is already taken, you'll have to come up with a different name.

After that you'll be presented with a prompt:

```
> |
```

Type a message and press Enter to send it.

Press `Ctrl + C` to quit.

You'll probably want to start multiple clients for them to talk to each other
instead of just a single one speaking into the void.

Now Go and Chat!

## Contributing

If you'd like to contribute, fork the repository and open a pull request to
the `main` branch.

Make sure both `go build` and `go test ./...` succeed.

You might also find the following resources useful:

- [Building terminal apps from Scratch](https://sahaj.dev/blog/tui-from-scratch)
- [Build Your Own Text Editor](https://viewsourcecode.org/snaptoken/kilo/)
  A tutorial walking you through building your own terminal text editor. It's in C.
  There's also Golang adaptation, from which you might find
  [Entering Raw mode](https://gokilo.github.io/entering-raw-mode.html) pretty
  useful to understand how to switch to the raw mode in Go.
- `man 3 termios`, in particular "Raw mode" section (at least on Linux), if you
  just want to see the flags that are considered to represent the "Raw mode". You
  can also
  [read it online](https://www.man7.org/linux/man-pages/man3/termios.3.html).
- [VT100 User Guide](https://vt100.net/docs/vt100-ug/)
  In particular, [Chapter 3. Programmer Information](https://vt100.net/docs/vt100-ug/chapter3.html)
- [ANSI escape sequences gist](https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797)
- [ANSI escape codes on Wikipedia](https://en.wikipedia.org/wiki/ANSI_escape_code)
- [On handling terminal resizing](https://stackoverflow.com/questions/31619962/vt100-ansi-escape-sequences-getting-screen-size-conditional-ansi)
