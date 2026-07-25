# Use Debian-based image (Bookworm)
FROM golang:1.26-bookworm

# Install the necessary system libraries for Gio/Wayland
RUN apt-get update && apt-get install -y \
    git \
    curl \
    build-essential \
    pkg-config \
    libwayland-dev \
    libwayland-egl-backend-dev \
    libxkbcommon-dev \
    libxkbcommon-x11-dev \
    libx11-dev \
    libx11-xcb-dev \
    libxcursor-dev \
    libxrandr-dev \
    libxinerama-dev \
    libxi-dev \
    libxfixes-dev \
    libvulkan-dev \
    libegl1-mesa-dev \
    libgles2-mesa-dev \
    fontconfig \
    fonts-dejavu-core \
    && rm -rf /var/lib/apt/lists/*

# Install Air
RUN curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

ENV GOCACHE=/tmp/go-cache
ENV GOMODCACHE=/tmp/go-mod

RUN mkdir -p /tmp/go-cache /tmp/go-mod && \
    chmod -R 777 /tmp/go-cache /tmp/go-mod

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .

# CMD ["air", "-c", ".air.toml"]
