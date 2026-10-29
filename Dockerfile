FROM golang:1.27.3@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:7bee201478b2f6266aae01243740ee85c16281e32f42d3aed9fdf04f4eace7a1
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]