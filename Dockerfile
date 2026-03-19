FROM golang:1.25

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o nrs-authentication ./cmd/ms-authentication

EXPOSE 8080

CMD ["./nrs-authentication"]
