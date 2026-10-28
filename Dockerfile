FROM golang:1.27.3@sha256:6f1acc898d064a4e9d5773ee574a9af6a9b786306939ac14d7fd806bf62046db 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:f76fce2964611eb1e25b90d906727884db12a8918a6d85a0df2619a9ef845c7c
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]