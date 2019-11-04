FROM golang:1.13

WORKDIR /go/src/app
COPY . .

RUN apt-get update
RUN apt-get install -y python3 python3-pip libfuse-dev
RUN pip3 install pytest

RUN go build ./cmd/filebox-server
RUN go build ./cmd/filebox-client

WORKDIR /go/src/app/test
CMD pytest