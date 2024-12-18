FROM postgres:15

# Environment variables for database configuration
ENV POSTGRES_USER=sarim
ENV POSTGRES_PASSWORD=1234
ENV POSTGRES_DB=vectors

# Expose the PostgreSQL port
EXPOSE 3399

# Custom PostgreSQL configuration
CMD ["postgres", "-c", "port=3399"]
