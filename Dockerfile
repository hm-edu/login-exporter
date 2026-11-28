FROM golang:1.27.4@sha256:5f1b98fa02fe396ebce2a0227a0e08309e11792915b9dd2e1f71053d7e7e6666 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:5bb2432d550896f4cb4266c367c48d82e6e2ed61f71ced68687bb175d62ce312
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]