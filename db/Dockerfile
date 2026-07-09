FROM timescale/timescaledb:latest-pg16

# Copy initialization script to docker entrypoint so it runs when DB starts
COPY init.sql /docker-entrypoint-initdb.d/01-init.sql
