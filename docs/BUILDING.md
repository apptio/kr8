# Building

kr8 is written in [Golang](https://golang.org). To build it, you'll need to perform a few steps.

## Install Golang

This varies depending on your operating system. There are instructions [here](https://golang.org/doc/install)

## Create a GOPATH

All go packages go into your `$GOPATH` so you'll need to create one.

```
mkdir -p $HOME/go/{src,bin}
export GOPATH=$HOME/go
export GOBIN=$HOME/go/bin
```

## Clone the repo

Go expects all packages to be in your `$GOPATH`. Clone this repo into your `$GOPATH`.

```
mkdir -p $GOPATH/src/github.com/apptio
git clone git@github.com.com:apptio/kr8.git $GOPATH/src/github.com/apptio
```

## Install glide

This project uses [glide](https://glide.sh) for its dependencies. Install it using the instructions detailed [here](http://glide.sh/)

## Install the dependencies

Now that everything is ready to go, you should be able to install all the dependencies. From within the repo, run:

```
glide install
```

This assumes that the previously installed `glide` binary is in your `$PATH`.

## Build the app!

now all the dependencies are installed, you can use the go build tool to build for your OS.

```
go build -o kr8 main.go
```

This will create a binary, `kr8`, in the current directory
