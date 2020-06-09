FROM php:7.4-apache
COPY --from=thijsdev/gcp-get-secret:latest /gcp-get-secret /

ADD index.php /var/www/html
ADD secret-docker-php-entrypoint /

ENTRYPOINT [ "/secret-docker-php-entrypoint" ]
