FROM postgres:17

COPY postgres/init-db.sql /tmp/
RUN cp /tmp/init-db.sql /docker-entrypoint-initdb.d/ && \
    chmod 644 /docker-entrypoint-initdb.d/init-db.sql && \
    rm /tmp/init-db.sql
