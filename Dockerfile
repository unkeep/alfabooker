FROM golang:1.11
WORKDIR /go/alfabooker
COPY . .
RUN go build ./...

FROM alpine:latest
WORKDIR /root/
COPY --from=0 /go/alfabooker/alfabooker .
CMD ["./alfabooker"]