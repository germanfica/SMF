# SMF 2.1.7

This directory builds a Docker image from the official `SimpleMachines/SMF`
tag `v2.1.7`. Jenkins fixes the SMF version, tag, and source SHA-256. The
database, settings, forum uploads, themes, packages, smileys, and cache are
kept in named Docker volumes.

The deployment assumes that Nginx Proxy Manager already uses the external
Docker network named `net`. Configure its proxy host to use `smf:80` on that
network. There is deliberately no host port published by this Compose project.
TLS must terminate at the proxy: the PHP configuration marks session cookies
as secure.

<img width="1354" height="756" alt="Screenshot from 2026-09-22 04-47-04" src="https://github.com/user-attachments/assets/20257e9e-04d4-4228-aa48-149136fa00d8" />

## Create archive

```bash
tar -czf ../smf.tar.gz . && mv ../smf.tar.gz ./smf.tar.gz
```

## Initial setup

Store these inventory variables with Ansible Vault. Use independently generated
passwords of at least 24 characters.

```yaml
SMF_DB_NAME: smf
SMF_DB_USER: smf
SMF_DB_PASSWORD: !vault |
  $ANSIBLE_VAULT;1.1;AES256
  ...
SMF_MYSQL_ROOT_PASSWORD: !vault |
  $ANSIBLE_VAULT;1.1;AES256
  ...
```

Run the Jenkins job once with `INSTALLER_MODE=enabled`. In the browser installer
use `database` as the database host, `smf` as both the database name and user,
and read the generated user password only on the target host:

```sh
sudo cat /opt/smf/secrets/mysql_password
```

As soon as the installer completes, run the same Jenkins job with
`INSTALLER_MODE=disabled`. The playbook removes the marker file and Apache
returns `403` for `install.php` and `upgrade.php`. Do not leave installer mode
enabled after setup.

## Local Compose use

Copy `.env.example` to `.env`, create `secrets/mysql_password` and
`secrets/mysql_root_password` with mode `0600`, then build and start it:

```sh
docker build \
  --build-arg SMF_VERSION=2.1.7 \
  --build-arg SMF_SOURCE_REF=v2.1.7 \
  --build-arg SMF_SOURCE_SHA256=2c9c0ea7df803ee03ff7755ea3651c680952e264b7c572439902bb18245c06a3 \
  --tag smf:2.1.7 .
docker compose create
docker compose run --rm --no-deps --entrypoint /bin/sh smf -ec \
  'touch /var/www/smf-config/installer-enabled && chmod 0600 /var/www/smf-config/installer-enabled'
docker compose up -d --wait
```

After finishing the browser installer, remove that marker and recreate the
forum container:

```sh
docker compose run --rm --no-deps --entrypoint /bin/sh smf -ec \
  'rm -f /var/www/smf-config/installer-enabled'
docker compose up -d --force-recreate smf
```

## Backups

Back up the MySQL volume and all `smf_*` volumes before upgrading SMF, MySQL,
themes, or packages. Test restores on a separate host. The image is intended
to keep forum code under the pipeline; add themes or packages by rebuilding
the image or by making a tested backup before using SMF's package manager.
Do not change the vaulted database passwords after MySQL has initialized unless
you rotate them inside MySQL in the same maintenance operation.
