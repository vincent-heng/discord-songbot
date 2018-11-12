FROM golang:alpine
RUN apk add --update --no-cache git
WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["app"]
