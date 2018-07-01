FROM golang:1.10.2-alpine3.7 AS builder

RUN apk update \
 && apk add git

RUN mkdir -p /go/src \
 && mkdir -p /go/bin \
 && mkdir -p /go/pkg

ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$PATH

RUN mkdir -p $GOPATH/src/app
ADD . $GOPATH/src/app

ADD . /go/src

WORKDIR $GOPATH/src/app

RUN go get .
RUN go get github.com/json-iterator/go
RUN CGO_ENABLED=0 go build -tags=jsoniter -a -installsuffix cgo -o /go/bin/server .

FROM txn2/n2pdf

WORKDIR /

COPY --from=builder /go/bin/server /server
ENTRYPOINT ["/server"]
