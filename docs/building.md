# Building kr8

As mentioned before, kr8 is coded in [Golang](https://golang.org/) so after following the next steps you should be able to use the kr8 executable like this:

`./kr8 --help`

## Prerequisites

1. Install and configure Go: https://golang.org/doc/install

2. Get familiar with Golang: https://golang.org/doc/

3. If you are fully testing the build you need to install: https://github.com/go-task/task

----

## Building the executable

On the project root:

`go build`

Go will start downloading the dependencies to create the executable, then you will end up with the following file:

`-rwxr-xr-x   1 myuser  mygroup    23M Apr  2 12:44 kr8`

Where `23M` is the current size of the executable.

## Troubleshooting the process

1. Dependencies download fail:
   There is a big number of reasons this could fail but the most important might be:
   -Networking problems: Check your connection to: github.com, golang.org and k8s.io.
   -Disk space: If no space is available on the disk, this step might fail.

2. The comand `go build` does not start the build:
   -Confirm you are in the correct project directory.
   -Make sure your go installation works: `go`

----
