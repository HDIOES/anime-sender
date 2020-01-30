FROM debian:stretch
COPY anime-sender settings.json webhook_cert.pem ./
ENTRYPOINT ["./anime-sender"]