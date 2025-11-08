FROM golang:1.25-alpine AS builder
RUN apk add --no-cache make git
WORKDIR /app

# Build the boot_handler Go application
COPY boot_handler/ ./boot_handler/
RUN make -C boot_handler build

FROM alpine:3.18
RUN apk update && apk add --no-cache dnsmasq supervisor jq inotify-tools gettext nginx

# Create directories for web content and TFTP
RUN mkdir /tftpboot
COPY tftpboot/* /tftpboot/

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY dnsmasq.conf.template /etc/dnsmasq.conf.template
COPY nginx.conf /etc/nginx/nginx.conf
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
COPY generate_dnsmasq_hosts.sh /usr/local/bin/generate_dnsmasq_hosts.sh
COPY watch_vms.sh /usr/local/bin/watch_vms.sh

RUN mkdir -p /www/initrds/x86_64 /www/initrds/aarch64
COPY initrd/output/x86_64/initrd.gz /www/initrds/x86_64/initrd.gz
COPY initrd/output/aarch64/initrd.gz /www/initrds/aarch64/initrd.gz

# Copy the boot handler and the multi-arch installer initrds from the builder stage
COPY --from=builder /app/boot_handler/bin/boot_handler /usr/local/bin/boot_handler
COPY boot_handler/boot.ipxe.go.template /boot.ipxe.go.template

RUN touch /var/log/dnsmasq.log && chmod 666 /var/log/dnsmasq.log
RUN touch /var/log/vm-watcher.log && chmod 666 /var/log/vm-watcher.log
RUN mkdir -p /var/log/nginx && touch /var/log/nginx/error.log && chmod -R 777 /var/log/nginx

EXPOSE 67/udp
EXPOSE 69/udp
EXPOSE 80/tcp
EXPOSE 53/udp
EXPOSE 53/tcp

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
