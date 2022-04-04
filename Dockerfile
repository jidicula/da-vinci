FROM golang:1.18.0-alpine as builder

RUN apk --no-cache add ca-certificates
WORKDIR /src/
COPY . /src/
RUN CGO_ENABLED=0 go build -buildvcs=false -o /da-vinci .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /da-vinci /da-vinci

CMD ["/da-vinci"]
