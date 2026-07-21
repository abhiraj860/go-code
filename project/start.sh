#!/bin/bash

docker compose down -v

docker compose up -d

echo "Waiting for services..."

echo "Services started!"

docker ps