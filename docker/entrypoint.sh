#!/bin/sh
set -eu

config_dir=/var/www/smf-config
template_dir=/usr/src/smf-config-template

for settings_file in Settings.php Settings_bak.php; do
	target_file="${config_dir}/${settings_file}"

	if [ ! -f "${target_file}" ]; then
		install -o www-data -g www-data -m 0660 "${template_dir}/${settings_file}" "${target_file}"
	else
		chown www-data:www-data "${target_file}"
		chmod 0660 "${target_file}"
	fi
done

exec docker-php-entrypoint "$@"
