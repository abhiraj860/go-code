#!/bin/bash

docker compose down -v


set -e

echo "Starting PostgreSQL and Kafka..."


docker compose up -d --remove-orphans


echo "Waiting for PostgreSQL..."
until docker exec postgres pg_isready -U admin -d appdb >/dev/null 2>&1; do
    sleep 1
done
echo "PostgreSQL is ready"

echo "Waiting for Kafka..."
until docker exec kafka /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server localhost:9092 \
    --list >/dev/null 2>&1; do
    sleep 1
done
echo "Kafka is ready"

echo "Creating Kafka topic..."
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server localhost:9092 \
    --create \
    --if-not-exists \
    --topic arbiter.incidents.created \
    --partitions 3 \
    --replication-factor 1

echo "PostgreSQL tables:"
docker exec postgres psql -U admin -d appdb -c "\dt"

echo "Kafka topics:"
docker exec kafka /opt/kafka/bin/kafka-topics.sh \
    --bootstrap-server localhost:9092 \
    --list

echo "Setup complete!"