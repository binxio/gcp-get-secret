FROM php:7.4-apache
COPY --from=binxio/gcp-get-secret:latest /gcp-get-secret /usr/bin/

ADD index.php /var/www/html

ENV PGPASSWORD=gcp:///gcp-get-secret-php-demo-pgpassword?default=postgres

ENTRYPOINT [ "/usr/bin/gcp-get-secret", "-verbose" ]
CMD [ "docker-php-entrypoint", "apache2-foreground" ]
