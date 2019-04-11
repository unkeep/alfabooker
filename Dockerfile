FROM golang:1.12-alpine
WORKDIR /go/alfabooker
COPY . .
RUN go build ./...

FROM alpine:latest
WORKDIR /root/
COPY --from=0 /go/alfabooker/alfabooker .
EXPOSE 8080
CMD ["./alfabooker"]