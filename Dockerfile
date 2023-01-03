FROM alpine
COPY pod /pod
WORKDIR "/"
CMD ["./pod"]
