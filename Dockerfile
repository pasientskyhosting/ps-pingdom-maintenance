FROM alpine:latest as certs
RUN apk --update add ca-certificates
FROM scratch
ENV PATH=/bin:/go/bin
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
# Copy our static executable.
COPY bin/ps-pingdom-maintenance64 /go/bin/ps-pingdom-maintenance
ENTRYPOINT ["/go/bin/ps-pingdom-maintenance"]
