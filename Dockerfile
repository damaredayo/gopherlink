FROM golang:1.17.2-buster

WORKDIR /go/src/app
COPY . .

RUN apt-get update && apt install -y libopusfile0 libopus0

EXPOSE 50051

RUN go build -o gopherlink
CMD ["./gopherlink"]