+++
title = "Contributing"
weight = 50
+++

# Building from Source

## Dependencies

kr8 uses [Go modules](https://github.com/golang/go/wiki/Modules) for its dependencies, so there's no longer any need to clone into your `$GOPATH`.

Simply clone into any directory and then download the modules:

```
git clone https://github.com/apptio/kr8.git
go mod download
```

## Building

Once you've downloaded all the dependencies, you should be able to build easily:

```
go build -o kr8
```

We use [goreleaser](https://goreleaser.com) for building the final binaries, so if you'd like to build a snapshot version, simple run:

```
goreleaser --skip-publish --snapshot --rm-dist
```

