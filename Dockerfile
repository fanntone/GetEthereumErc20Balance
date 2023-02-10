FROM golang
RUN mkdir -p /app
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o app
ENTRYPOINT ["./app"]
