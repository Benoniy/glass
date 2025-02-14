FROM golang:1.23.6-nanoserver

COPY "app" "/app"

WORKDIR "/app"
RUN go build ./glass.go

ENTRYPOINT [ "glass" ]