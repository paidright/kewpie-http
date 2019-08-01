FROM scratch

ADD ca-certificates.crt /etc/ssl/certs/
ADD zoneinfo.tar.gz /
ADD ./dist/linux-amd64/kewpie_http /kewpie_http

ENTRYPOINT ["/kewpie_http"]
