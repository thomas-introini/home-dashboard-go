# syntax=docker/dockerfile:1

FROM node:20 as tailwind

COPY . .

RUN npm install && \
    npm run build

FROM golang:1.22 as builder

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY . .

RUN wget https://github.com/benbjohnson/litestream/releases/download/v0.3.13/litestream-v0.3.13-linux-amd64.tar.gz \
    && tar xvf litestream-v0.3.13-linux-amd64.tar.gz

# Build (go-sqlite3 requires CGO_ENABLED=1)
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o /home-dashboard

# Second stage: setup the runtime environment
FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY litestream.yml litestream.yml
COPY static static/

COPY --from=builder /home-dashboard .
COPY --from=builder /app/litestream .

COPY --from=tailwind /static/css/main.css static/css/main.css

EXPOSE 8080

CMD /app/litestream restore -o sensor.db s3://$SENSOR_BUCKET/db && /app/litestream replicate -config /app/litestream.yml -exec "/app/home-dashboard"
# ENTRYPOINT [ "/app/home-dashboard" ]
