FROM golang:1.27.3@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:f76fce2964611eb1e25b90d906727884db12a8918a6d85a0df2619a9ef845c7c
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]