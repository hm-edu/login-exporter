FROM golang:1.27.4@sha256:3102e2c7eeee6e16de01ca3f35c56434696f70702a30b3d564b0244560981bb0 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:381353d3530ab9d579d2fe8b9c0f4519c995617f5a71daf6bb649e2d12005bd9
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]