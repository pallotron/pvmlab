FROM alpine:3.18
RUN apk update && apk add --no-cache dnsmasq supervisor jq inotify-tools gettext nginx

# Create directories for web content and TFTP
RUN mkdir /tftpboot
COPY tftpboot/* /tftpboot/
RUN chmod -R 755 /tftpboot

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY dnsmasq.conf.template /etc/dnsmasq.conf.template
COPY nginx.conf /etc/nginx/nginx.conf
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
COPY generate_dnsmasq_hosts.sh /usr/local/bin/generate_dnsmasq_hosts.sh
COPY watch_vms.sh /usr/local/bin/watch_vms.sh

RUN mkdir -p /www/initrds/x86_64 /www/initrds/aarch64
RUN chmod -R 755 /www
COPY initrd/output/x86_64/initrd.gz /www/initrds/x86_64/initrd.gz
COPY initrd/output/aarch64/initrd.gz /www/initrds/aarch64/initrd.gz
RUN chmod 644 /www/initrds/x86_64/initrd.gz /www/initrds/aarch64/initrd.gz

# Copy the boot handler and the multi-arch installer initrds
COPY boot_handler/bin/boot_handler_* /usr/local/bin/
RUN if [ "$(uname -m)" = "x86_64" ]; then \
        mv /usr/local/bin/boot_handler_amd64 /usr/local/bin/boot_handler; \
    elif [ "$(uname -m)" = "aarch64" ]; then \
        mv /usr/local/bin/boot_handler_arm64 /usr/local/bin/boot_handler; \
    fi
COPY boot_handler/boot.ipxe.go.template /boot.ipxe.go.template
RUN chmod 644 /boot.ipxe.go.template

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
