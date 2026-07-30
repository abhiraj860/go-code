-- One database per service: catalog, inventory and order own their schemas
-- independently, so no service can reach across a boundary with a JOIN.
-- Runs once, on first container start, via docker-entrypoint-initdb.d.

CREATE DATABASE ticketflow_catalog OWNER ticketflow;
CREATE DATABASE ticketflow_inventory OWNER ticketflow;
CREATE DATABASE ticketflow_order OWNER ticketflow;
