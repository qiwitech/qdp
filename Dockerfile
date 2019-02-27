FROM scratch
COPY plutos /
COPY plutoapi /
COPY plutosqldb /
COPY plutoclient /
ENTRYPOINT ["/plutos"]
EXPOSE 31337
