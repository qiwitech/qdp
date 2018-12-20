FROM ubuntu

ARG BINARY

COPY ${BINARY} /plutoapi

ENTRYPOINT ["/plutoapi", "-listen", ":80"]

EXPOSE 80
