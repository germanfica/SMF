#!/bin/sh
set -eu

config_dir=/var/www/smf-config
template_dir=/usr/src/smf-config-template

configure_default_settings_paths() {
	settings_file="$1"
	temporary_file="${settings_file}.tmp"

	awk '
		/^\$cachedir = dirname\(__FILE__\)/ {
			print "$cachedir = \047/var/www/html/cache\047;"
			next
		}
		/^\$boarddir = dirname\(__FILE__\);/ {
			print "$boarddir = \047/var/www/html\047;"
			next
		}
		/^\$sourcedir = dirname\(__FILE__\)/ {
			print "$sourcedir = \047/var/www/html/Sources\047;"
			next
		}
		/^\$packagesdir = dirname\(__FILE__\)/ {
			print "$packagesdir = \047/var/www/html/Packages\047;"
			next
		}
		{ print }
	' "${settings_file}" > "${temporary_file}"

	mv "${temporary_file}" "${settings_file}"
}

for settings_file in Settings.php Settings_bak.php; do
	target_file="${config_dir}/${settings_file}"

	if [ ! -f "${target_file}" ]; then
		install -o www-data -g www-data -m 0660 "${template_dir}/${settings_file}" "${target_file}"
	fi

	configure_default_settings_paths "${target_file}"
	chown www-data:www-data "${target_file}"
	chmod 0660 "${target_file}"
done

exec docker-php-entrypoint "$@"
