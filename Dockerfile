FROM golang:1.27.4@sha256:5e8afd493a7124a8586e134d9b532731161d4b9becf9dee5a5956b107426f93d 
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o exporter .
FROM chromedp/headless-shell@sha256:8c574e39f0db5c1d9efe7c472c6e0fcfc1d91ac0be70eb1dead0f2d52cf104cc
COPY --from=0 /app/exporter /app/exporter
ENTRYPOINT ["/app/exporter"]