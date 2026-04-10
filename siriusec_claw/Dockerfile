# Stage 1: Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/ui
COPY ui/package*.json ./
RUN npm ci
COPY ui/ ./
RUN npm run build

# Stage 2: Build backend
FROM golang:1.24-alpine AS backend
RUN apk add --no-cache git
WORKDIR /app/src
COPY src/go.mod src/go.sum ./
RUN go mod download
COPY src/ ./
COPY --from=frontend /app/src/embed/frontend/ ./embed/frontend/
ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
  -ldflags "-s -w -X github.com/siriusec/siriusec_claw/pkg/version.Version=${VERSION}" \
  -o /siriusec_claw \
  ./cmd/siriusec_claw

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 1000 claw
USER claw
WORKDIR /home/claw
COPY --from=backend /siriusec_claw /usr/local/bin/siriusec_claw
EXPOSE 3578
ENTRYPOINT ["siriusec_claw"]
CMD ["gateway", "--port", "3578"]
