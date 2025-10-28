FROM alpine:3.18
RUN apk update && apk add --no-cache dnsmasq supervisor jq inotify-tools gettext nginx

# Create TFTP root and copy iPXE binaries
RUN mkdir /tftpboot
COPY tftpboot/* /tftpboot/
COPY www /www/

COPY supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY dnsmasq.conf.template /etc/dnsmasq.conf.template
COPY nginx.conf /etc/nginx/nginx.conf
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
COPY generate_dnsmasq_hosts.sh /usr/local/bin/generate_dnsmasq_hosts.sh
COPY watch_vms.sh /usr/local/bin/watch_vms.sh

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