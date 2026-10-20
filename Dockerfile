FROM golang:1.27.2@sha256:820153096fda4de6ac1b7194d12e416b7dd8f4e9a33d17e0345b111b945162d8 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:53cc0f33c0598ff244f6e45458f6dfaf0ae4206d388ae7364903afd1d9d14f0a
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]