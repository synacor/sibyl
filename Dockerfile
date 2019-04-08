FROM golang:1.12 AS build
WORKDIR /build
COPY . /build
RUN go get github.com/GeertJohan/go.rice/... \
    && CGO_ENABLED=0 go build -o sibyl \
    && rice append --exec sibyl

FROM alpine:latest
COPY --from=build /build/sibyl /bin/sibyl
ENTRYPOINT [ "/bin/sibyl" ]
