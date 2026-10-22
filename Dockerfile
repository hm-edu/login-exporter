FROM golang:1.27.3@sha256:f20d83a83e18dbd2a4714b8ffd4b30b287b6243962dd75608e86abcc63ba4efc 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:53cc0f33c0598ff244f6e45458f6dfaf0ae4206d388ae7364903afd1d9d14f0a
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]