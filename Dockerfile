FROM golang:alpine AS build
WORKDIR /app
COPY . .
RUN go mod tidy && \
    echo "start building..." && \
    GOOS=linux GOARCH=amd64 go build -o /cloud-disk-backup .

FROM gcr.io/google.com/cloudsdktool/google-cloud-cli:alpine
RUN apk add --no-cache tzdata jq
ENV TZ=Asia/Taipei
COPY --from=build /cloud-disk-backup /cloud-disk-backup
RUN chmod +x /cloud-disk-backup
ENTRYPOINT ["/cloud-disk-backup"]