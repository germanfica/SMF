pipeline {
    agent { label 'build server' }

    parameters {
        choice(
            name: 'INSTALLER_MODE',
            choices: ['disabled', 'enabled'],
            description: 'Use enabled only for the first installation. Deploy disabled again as soon as setup finishes.'
        )
    }

    environment {
        SMF_VERSION = '2.1.7'
        SMF_SOURCE_REF = 'v2.1.7'
        SMF_SOURCE_SHA256 = '2c9c0ea7df803ee03ff7755ea3651c680952e264b7c572439902bb18245c06a3'
        SMF_IMAGE_NAME = 'smf'
        SMF_IMAGE_TAG = '2.1.7'
        SMF_MYSQL_IMAGE = 'mysql:8.4@sha256:0744ee5ef89ce6ccfa13de3e579fe6b9e27f93dd70da9c06d2c908b1b193fb8d'

        // Configure these names in Jenkins Credentials and adapt only the IDs.
        ANSIBLE_SSH_CREDENTIALS_ID = 'ansible-ssh-private-key'
        ANSIBLE_VAULT_CREDENTIALS_ID = 'ansible-vault-password'

        ANSIBLE_INVENTORY_PATH = '/opt/ansible-infra/inventories/production/applications/smf/inventory.yml'
        ANSIBLE_PLAYBOOK_PATH = '/opt/ansible-infra/playbooks/install-smf.yml'
        TARGET_HOSTS = 'smf'
        LIMIT_HOSTS = 'smf'
        SMF_PROXY_NETWORK = 'net'
    }

    options {
        disableConcurrentBuilds()
        timestamps()
        buildDiscarder(logRotator(numToKeepStr: '20', artifactNumToKeepStr: '20'))
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Validate deployment files') {
            steps {
                sh '''#!/bin/sh
                    set -eu
                    test "$SMF_VERSION" = '2.1.7'
                    test "$SMF_SOURCE_REF" = 'v2.1.7'
                    test "$SMF_SOURCE_SHA256" = '2c9c0ea7df803ee03ff7755ea3651c680952e264b7c572439902bb18245c06a3'
                    test -f Dockerfile
                    test -f docker/entrypoint.sh
                    test -f templates/docker-compose.smf.yml.j2
                    test -f playbooks/install-smf.yml
                    ansible-playbook --syntax-check playbooks/install-smf.yml
                '''
            }
        }

        stage('Build SMF image') {
            steps {
                sh '''#!/bin/sh
                    set -eu
                    docker build --pull \
                        --build-arg "SMF_VERSION=$SMF_VERSION" \
                        --build-arg "SMF_SOURCE_REF=$SMF_SOURCE_REF" \
                        --build-arg "SMF_SOURCE_SHA256=$SMF_SOURCE_SHA256" \
                        --tag "$SMF_IMAGE_NAME:$SMF_IMAGE_TAG" \
                        .
                '''
            }
        }

        stage('Verify SMF image') {
            steps {
                sh '''#!/bin/sh
                    set -eu
                    docker run --rm --entrypoint /bin/sh "$SMF_IMAGE_NAME:$SMF_IMAGE_TAG" -ec '
                        php -m | grep -qx "mbstring"
                        php -m | grep -qx "mysqli"
                        php -m | grep -qx "gd"
                        php -m | grep -qx "zip"
                        php -m | grep -qx "Zend OPcache"
                        apache2ctl -t
                        test -L /var/www/html/Settings.php
                        grep -F "SMF_VERSION" /var/www/html/install.php
                        grep -F "2.1.7" /var/www/html/install.php
                    '
                '''
            }
        }

        stage('Package SMF image') {
            steps {
                sh '''#!/bin/sh
                    set -eu
                    mkdir -p artifacts
                    docker save --output "artifacts/smf-v$SMF_VERSION.tar" "$SMF_IMAGE_NAME:$SMF_IMAGE_TAG"
                    sha256sum "artifacts/smf-v$SMF_VERSION.tar" > "artifacts/smf-v$SMF_VERSION.tar.sha256"
                '''
                archiveArtifacts artifacts: 'artifacts/*.sha256', fingerprint: true
            }
        }

        stage('Deploy') {
            steps {
                script {
                    def installerEnabled = params.INSTALLER_MODE == 'enabled' ? 'true' : 'false'

                    withCredentials([
                        file(credentialsId: env.ANSIBLE_VAULT_CREDENTIALS_ID, variable: 'VAULT_PASS_FILE'),
                        sshUserPrivateKey(credentialsId: env.ANSIBLE_SSH_CREDENTIALS_ID, keyFileVariable: 'SSH_KEY', usernameVariable: 'SSH_USER')
                    ]) {
                        withEnv(["SMF_INSTALLER_ENABLED=${installerEnabled}"]) {
                            sh '''#!/bin/sh
                                set -eu
                                ansible-playbook \
                                    -i "$ANSIBLE_INVENTORY_PATH" \
                                    --private-key="$SSH_KEY" \
                                    --vault-password-file="$VAULT_PASS_FILE" \
                                    "$ANSIBLE_PLAYBOOK_PATH" \
                                    --limit "$LIMIT_HOSTS" \
                                    --extra-vars "TARGET_HOSTS=$TARGET_HOSTS SMF_IMAGE_NAME=$SMF_IMAGE_NAME SMF_IMAGE_TAG=$SMF_IMAGE_TAG SMF_IMAGE_TAR=$WORKSPACE/artifacts/smf-v$SMF_VERSION.tar SMF_MYSQL_IMAGE=$SMF_MYSQL_IMAGE SMF_PROXY_NETWORK=$SMF_PROXY_NETWORK SMF_COMPOSE_TEMPLATE_SRC=$WORKSPACE/templates/docker-compose.smf.yml.j2 SMF_INSTALLER_ENABLED=$SMF_INSTALLER_ENABLED"
                            '''
                        }
                    }
                }
            }
        }
    }

    post {
        always {
            sh '''#!/bin/sh
                docker image rm "$SMF_IMAGE_NAME:$SMF_IMAGE_TAG" >/dev/null 2>&1 || true
            '''
        }
    }
}
