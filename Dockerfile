FROM golang:1.27.4@sha256:205c7636d74e0ffbe3d007162df7d74f859f5a8c8322dfb9f907977686c126cf 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:8c574e39f0db5c1d9efe7c472c6e0fcfc1d91ac0be70eb1dead0f2d52cf104cc
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]