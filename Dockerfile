FROM golang:1.21.5 
WORKDIR /app
COPY . .
RUN go build -o exporter .
FROM chromedp/headless-shell
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]