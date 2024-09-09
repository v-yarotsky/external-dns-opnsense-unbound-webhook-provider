FROM scratch

COPY external-dns-opnsense-unbound-webhook-provider /

EXPOSE 8888
ENTRYPOINT ["/external-dns-opnsense-unbound-webhook-provider"]
