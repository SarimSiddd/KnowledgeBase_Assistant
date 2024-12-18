FROM postgres:15

# Environment variables for database configuration
ENV POSTGRES_USER=sarim
ENV POSTGRES_PASSWORD=1234
ENV POSTGRES_DB=vectors

# Copy custom PostgreSQL configuration
COPY postgresql.conf /etc/postgresql/postgresql.conf

# Expose the PostgreSQL port
EXPOSE 3399

# Start PostgreSQL with custom config
CMD ["postgres", "-c", "config_file=/etc/postgresql/postgresql.conf"]
