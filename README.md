<p align="center" style="margin-bottom: 0px !important;"> 
    <img src="https://cdn.discordapp.com/attachments/720202948797399079/973707818202976286/gomusic_copy.png" width=200>
</p>
<h1 align="center" style="font-size:48px">Gopherlink</h1>

<div align="center">

### Gopherlink is a Audio Streaming application/framework for Discord designed for ease of use and performance.
</div>

# 
## Supported Sources
- Youtube
- More to come
#
## Installation
Gopherlink is designed to run on Linux. Windows + Mac OS are untested.

[Docker](https://www.docker.com) is the preffered method of installation, you can use the application this way via the provided `Dockerfile`.

If you do not wish to use Docker, you can use the application the following way.

### Dependencies 

In Debian based systems this can be done with

    apt-get update && apt-get install -y libopus-dev libopusfile-dev ffmpeg python3

You will have to aquire Go yourself as the Debian repositories do not have it.

You will also have to aquire an executable for `yt-dlp`, this can be found [here](https://github.com/yt-dlp/yt-dlp/releases/).

In Arch **based** systems you can do this easily via an AUR helper like [yay](https://github.com/Jguer/yay).

    yay -Syy opus opusfile ffmpeg python3 yt-dlp go

### Building

Gopherlink is written in Go (surprisingly enough) and can be built very easily.

First if you haven't already, you must clone the repository with

    git clone https://github.com/damaredayo/gopherlink.git

Then, you will want to enter the directory with `cd gopherlink/app` at which point you can run

    go build -o gopherlink

Which will build the application, it can then be ran via `./gopherlink`

### Environment Variables
There is currently one environment variable which is the path of `yt-dlp`.

The name of this environment variable is `YTDL_PATH`. 

**Gopherlink will currently NOT work without this.**

## Usage

Gopherlink uses [GRPC](https://grpc.io/) currently to communicate over a network. There ARE plans to move away from this (as I hate protobuf with a burning passion).

The protobuf files can be found at https://github.com/damaredayo/gopherlink/tree/master/proto

If you do not wish to use GRPC (I don't blame you) then you can use Gopherlink like a go package instead without having to build the application.

This can be easily done with `go get` by doing

    go get -u github.com/damaredayo/gopherlink


The functions are fairly idiomatic, if you have worked with Lavalink in the past then the basic flow will feel very familiar.

Examples of implementation with both methods will come soon. (I will likely move away from GRPC first.)
