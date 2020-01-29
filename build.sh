#!/bin/bash

dep ensure
go build -o anime-sender
docker build -t ivantimofeev/anime-sender .
docker push ivantimofeev/anime-sender