FROM golang:1.27.3@sha256:e519d3b0a5b285e5aa53ab9eef02bc6bbb9cb04608389b743a136a9ce75f0d1e 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:830fa77e0602cb42e5cb2cbd9237ba27b649ad69d5fb82b54713aac8b7ab4f29
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]