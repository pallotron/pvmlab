FROM alpine:latest

FROM alpine:latest as ipxe-builder

RUN apk update && apk add --no-cache alpine-ipxe
RUN mkdir /ipxe-binaries
ARG TARGETARCH
RUN if [ "$TARGETARCH" = "amd64" ]; then cp /usr/share/alpine-ipxe/ipxe.efi /ipxe-binaries/ipxe.efi; fi
RUN if [ "$TARGETARCH" = "arm64" ]; then cp /usr/share/alpine-ipxe/snp.efi /ipxe-binaries/ipxe-arm64.efi; fi

FROM alpine:latest

RUN apk update && apk add --no-cache dnsmasq supervisor

# Create TFTP root and copy iPXE binaries
RUN mkdir /tftpboot
COPY --from=ipxe-builder /ipxe-binaries/* /tftpboot/

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY dnsmasq.conf /etc/dnsmasq.conf



EXPOSE 67/udp
EXPOSE 69/udp

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
