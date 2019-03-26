FROM php:7.2-cli

RUN apt-get update && apt-get install -y -q zip unzip zlib1g-dev && apt-get clean

RUN cd /usr/local/bin && curl -sS https://getcomposer.org/installer | php
RUN cd /usr/local/bin && mv composer.phar composer

RUN pecl install grpc
RUN echo "extension=grpc.so" > /usr/local/etc/php/conf.d/30-grpc.ini

RUN pecl install protobuf
RUN echo "extension=protobuf.so" > /usr/local/etc/php/conf.d/30-protobuf.ini

RUN mkdir -p /usr/src/open-match/examples/functions/php/mmlogic-simple
COPY examples/functions/php/mmlogic-simple /usr/src/open-match/examples/functions/php/mmlogic-simple
COPY config /usr/src/open-match/examples/functions/php/mmlogic-simple/config
WORKDIR /usr/src/open-match/examples/functions/php/mmlogic-simple

RUN composer install

CMD [ "php", "./harness.php" ]
