FROM debian:stretch
COPY anime-sender settings.json ./
ENTRYPOINT ["./anime-sender"]