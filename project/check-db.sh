#!/bin/bash

echo "Tables:"
docker exec postgres psql -U admin -d appdb -c "\dt"

echo
echo "Users:"
docker exec postgres psql -U admin -d appdb -c "SELECT * FROM users;"