FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src/backend

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /bin/app ./cmd/app

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

WORKDIR /bin

COPY --from=builder /bin/app .

ENV ADDR=0.0.0.0:3000
ENV GO_ENV=production

EXPOSE 3000

CMD ["/bin/app"]