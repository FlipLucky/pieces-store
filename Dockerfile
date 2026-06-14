FROM golang:1.26-bookworm

RUN apt-get update && apt-get install -y git curl build-essential && rm -rf /var/lib/apt/lists/*

# Install Air for live reloading
RUN curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Install Goose for migrations
# RUN GOBIN=/usr/local/bin go install github.com/pressly/goose/v3/cmd/goose@v3.20.0

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Run Air
CMD ["air", "-c", ".air.toml"]
