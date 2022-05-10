FROM golang:1.17.2-buster

WORKDIR /go/src/app/gopherlink
COPY . .

RUN apt-get update && apt-get install -y libopus-dev libopusfile-dev ffmpeg python3

EXPOSE 50051

WORKDIR /go/src/app/gopherlink/app
RUN curl -LJO https://github.com/yt-dlp/yt-dlp/releases/download/2022.04.08/yt-dlp
RUN chmod +x yt-dlp
RUN go build -o gopherlink
RUN chmod +x gopherlink
CMD ["./gopherlink"]
