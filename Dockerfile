FROM node:20-alpine AS tailwind
WORKDIR /app
ADD . .
RUN apk update && apk upgrade && apk add make && \
  make tailwind-server

FROM golang:1.23-alpine AS builder
ARG COMPONENT=ngrokd
ENV GO111MODULE=on
ENV CGO_ENABLED=0 
ENV GOOS=linux
WORKDIR /app
RUN apk update && apk upgrade && apk add --no-cache ca-certificates make && \
  update-ca-certificates
COPY --from=tailwind /app .
RUN echo "Execution for $COMPONENT " && make release-server && chmod +x /app/bin/server/${COMPONENT}

FROM scratch
ARG COMPONENT=ngrokd
COPY --from=builder "/app/bin/server/${COMPONENT}" /${COMPONENT}
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/ngrokd"]
