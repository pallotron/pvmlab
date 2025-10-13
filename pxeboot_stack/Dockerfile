FROM alpine:3.18

FROM alpine:3.18 as ipxe-builder

RUN apk update && apk add --no-cache alpine-ipxe
RUN mkdir /ipxe-binaries
ARG TARGETARCH
RUN if [ "$TARGETARCH" = "amd64" ]; then cp /usr/share/alpine-ipxe/ipxe.efi /ipxe-binaries/ipxe.efi; fi
RUN if [ "$TARGETARCH" = "arm64" ]; then cp /usr/share/alpine-ipxe/snp.efi /ipxe-binaries/ipxe-arm64.efi; fi

FROM alpine:3.18

RUN apk update && apk add --no-cache dnsmasq supervisor jq inotify-tools

# Create TFTP root and copy iPXE binaries
RUN mkdir /tftpboot
COPY --from=ipxe-builder /ipxe-binaries/* /tftpboot/

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY dnsmasq.conf /etc/dnsmasq.conf
COPY generate_dnsmasq_hosts.sh /usr/local/bin/generate_dnsmasq_hosts.sh
COPY watch_vms.sh /usr/local/bin/watch_vms.sh

RUN touch /var/log/dnsmasq.log && chmod 666 /var/log/dnsmasq.log
RUN touch /var/log/vm-watcher.log && chmod 666 /var/log/vm-watcher.log

EXPOSE 67/udp
EXPOSE 69/udp

CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
