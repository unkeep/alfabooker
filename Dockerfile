FROM golang:1.12-stretch
WORKDIR /go/alfabooker
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build ./...

FROM alpine:latest
WORKDIR /root/
COPY --from=0 /go/alfabooker/alfabooker .
EXPOSE 8080
CMD ["./alfabooker"]