FROM golang:1.27.4@sha256:fe517acaa580cc7915457080f850c37e618b3c1e750f98b58fc6429b605147a1 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:8c574e39f0db5c1d9efe7c472c6e0fcfc1d91ac0be70eb1dead0f2d52cf104cc
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]