FROM ubuntu

ARG BINARY

COPY ${BINARY} /sqldb

ENTRYPOINT ["/sqldb"]
CMD ["-dbaddr", "mysql:3306"]

EXPOSE 38388
