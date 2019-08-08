FROM alpine:latest as certs
RUN apk --update add ca-certificates tzdata

FROM scratch

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=certs /usr/share/zoneinfo/ /usr/share/zoneinfo/
ADD ./dist/linux-amd64/kewpie_http /kewpie_http

ENTRYPOINT ["/kewpie_http"]
