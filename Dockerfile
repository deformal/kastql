# ── Stage 1: frontend build ───────────────────────────────────────────────────
FROM node:22-alpine AS web-builder

WORKDIR /web

# Install deps first so this layer is cached when only source changes.
COPY web/package.json web/package-lock.json ./
RUN npm ci --silent

COPY web/ ./
RUN npm run build

# ── Stage 2: Go build ─────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS go-builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Copy compiled frontend assets into the embed directory.
COPY --from=web-builder /web/dist ./internal/playground/dist

# CGO_ENABLED=0: modernc.org/sqlite is pure Go — no C toolchain needed.
RUN CGO_ENABLED=0 GOOS=linux go build \
      -ldflags="-s -w" \
      -trimpath \
      -o /kastql \
      ./cmd/kastql

# ── Stage 3: runtime ──────────────────────────────────────────────────────────
# distroless/static has no shell, no package manager — minimal attack surface.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=go-builder /kastql /kastql

VOLUME ["/data", "/etc/kastql"]

EXPOSE 8080

ENTRYPOINT ["/kastql"]
CMD ["-config", "/etc/kastql/config.yaml"]
