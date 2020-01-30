FROM debian:stretch
COPY anime-sender ./
COPY settings.json ./
COPY public.pem ./
ENTRYPOINT ["./anime-sender"]