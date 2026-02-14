FROM scratch

ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/external-dns-opnsense-unbound-webhook-provider /

EXPOSE 8888
ENTRYPOINT ["/external-dns-opnsense-unbound-webhook-provider"]
