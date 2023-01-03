FROM alpine
ADD pod /
ENTRYPOINT ["/pod"]
