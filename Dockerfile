FROM golang:1.27.3@sha256:921f61120011c774a7b37fd80bd3c3d259636090d3cd002b6a344322a0328082 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:f76fce2964611eb1e25b90d906727884db12a8918a6d85a0df2619a9ef845c7c
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]