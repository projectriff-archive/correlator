FROM golang:1.10 as builder

ARG PACKAGE=github.com/markfisher/correlator
ARG COMMAND=correlator

WORKDIR /go/src/${PACKAGE}
COPY vendor/github.com /go/src/github.com
COPY . .

RUN CGO_ENABLED=0 go build -v -a -installsuffix cgo ${COMMAND}.go

###########

FROM scratch

ARG PACKAGE=github.com/markfisher/correlator
ARG COMMAND=correlator
COPY --from=builder /go/src/${PACKAGE}/${COMMAND} /${COMMAND}

ENTRYPOINT ["/correlator"]
