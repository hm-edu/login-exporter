FROM golang:1.27.3@sha256:c292963ad0e5f51e843c726edc204a4e4c26a859ba1e4d736d91028aa3633048 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:7bee201478b2f6266aae01243740ee85c16281e32f42d3aed9fdf04f4eace7a1
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]