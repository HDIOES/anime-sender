FROM debian:stretch
COPY anime-sender ./
COPY settings.json ./
COPY webhook_cert.pem ./
ENTRYPOINT ["./anime-sender"]