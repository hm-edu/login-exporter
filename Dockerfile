FROM golang:1.27.4@sha256:fe517acaa580cc7915457080f850c37e618b3c1e750f98b58fc6429b605147a1 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:830fa77e0602cb42e5cb2cbd9237ba27b649ad69d5fb82b54713aac8b7ab4f29
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]