# Copyright 2019 Google LLC
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#     http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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

# TODO: Remove embedding the config once loading configuration is normalized and this can respect the ConfigMap.
COPY config /usr/src/open-match/examples/functions/php/mmlogic-simple/config
WORKDIR /usr/src/open-match/examples/functions/php/mmlogic-simple

RUN composer install

CMD [ "php", "./harness.php" ]
