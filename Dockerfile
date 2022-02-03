FROM node:17-alpine AS asset-builder

WORKDIR /usr/src/app
COPY frontend/package*.json ./
RUN npm ci
COPY frontend .
RUN npm run-script build

# ================================================================================

FROM golang:1.17.6-alpine AS app-builder

WORKDIR /usr/src/app

ENV GO111MODULE=on

RUN apk add -U --no-cache ca-certificates && update-ca-certificates

COPY go.mod .
COPY go.sum .

RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -a -ldflags "-s -w" -o /go/bin/poddy

# # ================================================================================

FROM scratch

WORKDIR /usr/src/app
ENV GIN_MODE=release

COPY --from=app-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=asset-builder /usr/src/app/dist frontend/dist
COPY --from=app-builder /go/bin/poddy .

ENTRYPOINT ["/usr/src/app/poddy"]