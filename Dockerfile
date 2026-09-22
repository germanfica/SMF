# syntax=docker/dockerfile:1.7

# This is an OCI index digest, so Docker selects the matching Linux platform.
ARG PHP_IMAGE=php:8.4-apache-bookworm@sha256:0629e7852d88a0939841973910d94157de7cd68e48acefac44546f8534b45d31
FROM ${PHP_IMAGE}

ARG SMF_VERSION
ARG SMF_SOURCE_REF
ARG SMF_SOURCE_SHA256

LABEL org.opencontainers.image.title="SMF" \
	org.opencontainers.image.description="Simple Machines Forum production image" \
	org.opencontainers.image.source="https://github.com/SimpleMachines/SMF" \
	org.opencontainers.image.licenses="BSD-3-Clause" \
	org.opencontainers.image.version="${SMF_VERSION}"

ENV APACHE_DOCUMENT_ROOT=/var/www/html \
	SMF_VERSION=${SMF_VERSION}

RUN set -eux; \
	test -n "${SMF_VERSION}"; \
	test -n "${SMF_SOURCE_REF}"; \
	test -n "${SMF_SOURCE_SHA256}"; \
	apt-get update; \
	apt-get install -y --no-install-recommends \
		ca-certificates \
		curl \
		libfreetype6-dev \
		libjpeg62-turbo-dev \
		libonig-dev \
		libpng-dev \
		libwebp-dev \
		libzip-dev; \
	docker-php-ext-configure gd --with-freetype --with-jpeg --with-webp; \
	docker-php-ext-install -j "$(nproc)" gd mbstring mysqli opcache zip; \
	a2enmod expires headers rewrite; \
	rm -rf /var/lib/apt/lists/*

RUN set -eux; \
	mkdir -p /tmp/smf-source /usr/src/smf-config-template /var/www/smf-config; \
	curl --fail --location --retry 5 --retry-all-errors --output /tmp/smf.tar.gz \
		"https://github.com/SimpleMachines/SMF/archive/refs/tags/${SMF_SOURCE_REF}.tar.gz"; \
	echo "${SMF_SOURCE_SHA256}  /tmp/smf.tar.gz" | sha256sum --check --strict; \
	tar --extract --gzip --file /tmp/smf.tar.gz --strip-components=1 --directory /tmp/smf-source; \
	grep --fixed-strings "define('SMF_VERSION', '${SMF_VERSION}');" /tmp/smf-source/other/install.php; \
	cp -a /tmp/smf-source/. /var/www/html/; \
	install -m 0640 /tmp/smf-source/other/Settings.php /usr/src/smf-config-template/Settings.php; \
	install -m 0640 /tmp/smf-source/other/Settings_bak.php /usr/src/smf-config-template/Settings_bak.php; \
	install -m 0644 /tmp/smf-source/other/install.php /var/www/html/install.php; \
	install -m 0644 /tmp/smf-source/other/install_2-1_mysql.sql /var/www/html/install_2-1_mysql.sql; \
	install -m 0644 /tmp/smf-source/other/install_2-1_postgresql.sql /var/www/html/install_2-1_postgresql.sql; \
	rm -rf /var/www/html/other /tmp/smf-source /tmp/smf.tar.gz; \
	ln -s /var/www/smf-config/Settings.php /var/www/html/Settings.php; \
	ln -s /var/www/smf-config/Settings_bak.php /var/www/html/Settings_bak.php; \
	chown -R root:root /var/www/html; \
	chown -R www-data:www-data \
		/var/www/html/attachments \
		/var/www/html/avatars \
		/var/www/html/cache \
		/var/www/html/Packages \
		/var/www/html/Smileys \
		/var/www/html/Themes; \
	find /var/www/html -type d -exec chmod 0755 {} +; \
	find /var/www/html -type f -exec chmod 0644 {} +

COPY docker/apache-smf.conf /etc/apache2/conf-available/smf.conf
COPY docker/php-production.ini /usr/local/etc/php/conf.d/zz-smf-production.ini
COPY docker/entrypoint.sh /usr/local/bin/smf-entrypoint

RUN set -eux; \
	a2enconf smf; \
	chmod 0755 /usr/local/bin/smf-entrypoint

ENTRYPOINT ["smf-entrypoint"]
CMD ["apache2-foreground"]
