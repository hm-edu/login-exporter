FROM golang:1.27.4@sha256:5f1b98fa02fe396ebce2a0227a0e08309e11792915b9dd2e1f71053d7e7e6666 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:381353d3530ab9d579d2fe8b9c0f4519c995617f5a71daf6bb649e2d12005bd9
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]