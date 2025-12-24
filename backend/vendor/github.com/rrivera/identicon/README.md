# IdentIcon


![CircleCI](https://img.shields.io/circleci/project/github/RedSparr0w/node-csgo-parser.svg)
 [![Go Report Card](https://goreportcard.com/badge/github.com/rrivera/identicon)](https://goreportcard.com/report/github.com/rrivera/identicon) [![](https://godoc.org/github.com/rrivera/identicon?status.svg)](http://godoc.org/github.com/rrivera/identicon)

**IdentIcon** is an open source avatar generator inspired by GitHub avatars. 


IdentIcon uses a deterministic algorithm that generates an image (using Golang's stdlib image encoders) based on a text (Generally Usernames, Emails or just random strings), by hashing it and iterating over the bytes of the digest to pick whether to draw a point, pick a color or choose where to go next.


IdentIcon's Generator enables the creation of customized figures: (NxN size, points density, custom color palette) as well as multiple exporting formats in case the developers want to generate their own images.

## Installation
```bash
$ go get github.com/rrivera/identicon
```

## Usage 

```go

import (
    "os"

    "github.com/rrivera/identicon"
)

// New Generator: Rehuse 
ig, err := identicon.New(
    "github", // Namespace
    5,        // Number of blocks (Size)
    3,        // Density
)

if err != nil {
    panic(err) // Invalid Size or Density
}

username := "rrivera"      // Text - decides the resulting figure
ii, err := ig.Draw(username) // Generate an IdentIcon

if err != nil {
    panic(err) // Text is empty
}

// File writer
img, _ := os.Create("icon.png")
defer img.Close()
// Takes the size in pixels and any io.Writer
ii.Png(300, img) // 300px * 300px

```

## Examples

### 5x5
|rrivera                                        | johndoe                                       | abc123                                      | modulo                                      |
:------------------------------------------------:|:---------------------------------------------:|:-------------------------------------------:|:--------------------------------------------|
![rrivera](./examples/5x5/rrivera.png)        | ![johndoe](./examples/5x5/johndoe.png)        | ![abc123](./examples/5x5/abc123.png)        | ![modulo](./examples/5x5/modulo.png)        |
![rrivera](./examples/5x5/rrivera_itx.png)    | ![johndoe](./examples/5x5/johndoe_itx.png)    | ![abc123](./examples/5x5/abc123_itx.png)    | ![modulo](./examples/5x5/modulo_itx.png)    |
![rrivera](./examples/5x5/rrivera_github.png) | ![johndoe](./examples/5x5/johndoe_github.png) | ![abc123](./examples/5x5/abc123_github.png) | ![modulo](./examples/5x5/modulo_github.png) |

### 7x7
|rrivera                                        |  johndoe                                      |  abc123                                     |  modulo                                      |
:------------------------------------------------:|:---------------------------------------------:|:-------------------------------------------:|:---------------------------------------------|
![rrivera](./examples/7x7/rrivera.png)        | ![johndoe](./examples/7x7/johndoe.png)        | ![abc123](./examples/7x7/abc123.png)        | ![modulo](./examples/7x7/modulo.png)         |
![rrivera](./examples/7x7/rrivera_itx.png)    | ![johndoe](./examples/7x7/johndoe_itx.png)    | ![abc123](./examples/7x7/abc123_itx.png)    | ![modulo](./examples/7x7/modulo_itx.png)     |
![rrivera](./examples/7x7/rrivera_github.png) | ![johndoe](./examples/7x7/johndoe_github.png) | ![abc123](./examples/7x7/abc123_github.png) | ![modulo](./examples/7x7/modulo_github.png)  |
           
### 10x10
|rrivera                                          |  johndoe                                        |  abc123                                       |  modulo                                       |
:--------------------------------------------------:|:-----------------------------------------------:|:---------------------------------------------:|:----------------------------------------------|
![rrivera](./examples/10x10/rrivera.png)        | ![johndoe](./examples/10x10/johndoe.png)        | ![abc123](./examples/10x10/abc123.png)        | ![modulo](./examples/10x10/modulo.png)        |
![rrivera](./examples/10x10/rrivera_itx.png)    | ![johndoe](./examples/10x10/johndoe_itx.png)    | ![abc123](./examples/10x10/abc123_itx.png)    | ![modulo](./examples/10x10/modulo_itx.png)    |
![rrivera](./examples/10x10/rrivera_github.png) | ![johndoe](./examples/10x10/johndoe_github.png) | ![abc123](./examples/10x10/abc123_github.png) | ![modulo](./examples/10x10/modulo_github.png) |

[View examples](./examples)

## Documentation

## Changelog

## Contribution

## License
MIT

Copyright (c) 2018-present, Ruben Rivera

