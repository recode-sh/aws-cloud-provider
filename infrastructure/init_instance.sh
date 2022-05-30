#!/bin/bash
# 
# Recode instance init.
# 
# This is the first script to run during development environment creation (via cloud-init).
# 
# See: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html
# 
# In a nutshell, this script:
# - create and configure the user "recode" (notably the SSH access)
# - configure and install the recode agent
#
# The next steps are assured by the recode agent via GRPC through SSH.
set -euo pipefail

log () {
  echo -e "${1}" >&2
}

log "\n\n"
log "---- Recode instance init (start) ----"
log "\n\n"

# Remove "debconf: unable to initialize frontend: Dialog" warnings
echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections

# We use "jq" in our exit trap and "curl" to download the recode agent
apt-get --assume-yes --quiet --quiet install jq curl

constructExitJSONResponse () {
  JSON_RESPONSE=$(jq --null-input \
  --arg exitCode "${1}" \
  --arg sshHostKeys "${2}" \
  '{"exit_code": $exitCode, "ssh_host_keys": $sshHostKeys}')

  echo "${JSON_RESPONSE}"
}

RECODE_SSH_SERVER_HOST_KEY_FILE_PATH="/home/recode/.ssh/recode_ssh_server_host_key.pub"
RECODE_INIT_RESULTS_FILE_PATH="/tmp/recode_init_results"

handleExit () {
  EXIT_CODE=$?

  rm --force "${RECODE_INIT_RESULTS_FILE_PATH}"

  log "\n\n"
  if [[ "${EXIT_CODE}" != 0 ]]; then
    constructExitJSONResponse "${EXIT_CODE}" "" >> "${RECODE_INIT_RESULTS_FILE_PATH}"
    log "---- Recode instance init (failed) (exit code ${EXIT_CODE}) ----"
  else
    SSH_HOST_KEYS="$(cat "${RECODE_SSH_SERVER_HOST_KEY_FILE_PATH}")"
    constructExitJSONResponse "${EXIT_CODE}" "${SSH_HOST_KEYS}" >> "${RECODE_INIT_RESULTS_FILE_PATH}"
    
    log "---- Recode instance init (success) ----"
  fi
  log "\n\n"

  exit "${EXIT_CODE}"
}

trap "handleExit" EXIT

# Lookup instance architecture for the recode agent
INSTANCE_ARCH=""
case $(uname -m) in
  i386)       INSTANCE_ARCH="386" ;;
  i686)       INSTANCE_ARCH="386" ;;
  x86_64)     INSTANCE_ARCH="amd64" ;;
  arm)        dpkg --print-architecture | grep -q "arm64" && INSTANCE_ARCH="arm64" || INSTANCE_ARCH="armv6" ;;
  aarch64_be) INSTANCE_ARCH="arm64" ;;
  aarch64)    INSTANCE_ARCH="arm64" ;;
  armv8b)     INSTANCE_ARCH="arm64" ;;
  armv8l)     INSTANCE_ARCH="arm64" ;;
esac

# -- Create / Configure the user "recode"

log "Creating user \"recode\""

RECODE_USER_HOME_DIR="/home/recode"
RECODE_USER_WORKSPACE_DIR="${RECODE_USER_HOME_DIR}/workspace"
RECODE_USER_WORKSPACE_CONFIG_DIR="${RECODE_USER_HOME_DIR}/.workspace-config"

groupadd --force recode
id -u recode >/dev/null 2>&1 || useradd --gid recode --home "${RECODE_USER_HOME_DIR}" --create-home --shell /bin/bash recode

# Let the user "recode" and the recode agent
# run docker commands without "sudo".
# See https://docs.docker.com/engine/install/linux-postinstall/
groupadd --force docker
usermod --append --groups docker recode

if [[ ! -f "/etc/sudoers.d/recode" ]]; then
  echo "recode ALL=(ALL) NOPASSWD:ALL" | tee /etc/sudoers.d/recode > /dev/null
fi

mkdir --parents "${RECODE_USER_WORKSPACE_DIR}"
mkdir --parents "${RECODE_USER_WORKSPACE_CONFIG_DIR}"
chown --recursive recode:recode "${RECODE_USER_HOME_DIR}"

log "Configuring home directory for user \"recode\""

# We want the user "recode" to be able to 
# connect through SSH via the generated SSH key.
# See below.
INSTANCE_SSH_PUBLIC_KEY="$(cat /home/ubuntu/.ssh/authorized_keys)"

# Run as "recode"
sudo --set-home --login --user recode -- env \
	INSTANCE_SSH_PUBLIC_KEY="${INSTANCE_SSH_PUBLIC_KEY}" \
bash << 'EOF'

mkdir --parents .ssh
chmod 700 .ssh

if [[ ! -f ".ssh/recode_ssh_server_host_key" ]]; then
  ssh-keygen -t ed25519 -f .ssh/recode_ssh_server_host_key -q -N ""
fi

chmod 644 .ssh/recode_ssh_server_host_key.pub
chmod 600 .ssh/recode_ssh_server_host_key

if [[ ! -f ".ssh/authorized_keys" ]]; then
  echo "${INSTANCE_SSH_PUBLIC_KEY}" >> .ssh/authorized_keys
fi

chmod 600 .ssh/authorized_keys

EOF

# -- Install the recode agent
#
# /!\ the SSH server host key ("recode_ssh_server_host_key") 
#     needs to be generated. See above.
#
# /!\ the user "recode" needs to be able to access 
#     the docker daemon. See above.

log "Installing the recode agent"

RECODE_AGENT_VERSION="0.1.0"
RECODE_AGENT_TMP_ARCHIVE_PATH="/tmp/recode-agent.tar.gz"
RECODE_AGENT_NAME="recode_agent"
RECODE_AGENT_DIR="/usr/local/bin"
RECODE_AGENT_PATH="${RECODE_AGENT_DIR}/${RECODE_AGENT_NAME}"
RECODE_AGENT_SYSTEMD_SERVICE_NAME="recode_agent.service"

if [[ ! -f "${RECODE_AGENT_PATH}" ]]; then
  rm --recursive --force "${RECODE_AGENT_TMP_ARCHIVE_PATH}"
  curl --fail --silent --show-error --location --header "Accept: application/octet-stream" "https://github.com/recode-sh/agent/releases/download/v${RECODE_AGENT_VERSION}/agent_${RECODE_AGENT_VERSION}_linux_${INSTANCE_ARCH}.tar.gz" --output "${RECODE_AGENT_TMP_ARCHIVE_PATH}"
  tar --directory "${RECODE_AGENT_DIR}" --extract --file "${RECODE_AGENT_TMP_ARCHIVE_PATH}"
  rm --recursive --force "${RECODE_AGENT_TMP_ARCHIVE_PATH}"
fi

chmod +x "${RECODE_AGENT_PATH}"

if [[ ! -f "/etc/systemd/system/${RECODE_AGENT_SYSTEMD_SERVICE_NAME}" ]]; then
  tee /etc/systemd/system/"${RECODE_AGENT_SYSTEMD_SERVICE_NAME}" > /dev/null << EOF
  [Unit]
  Description=This agent is used to establish connection with the Recode CLI.

  [Service]
  Type=simple
  ExecStart=${RECODE_AGENT_PATH}
  WorkingDirectory=${RECODE_AGENT_DIR}
  Restart=always
  User=recode
  Group=recode

  [Install]
  WantedBy=multi-user.target
EOF
fi

systemctl enable "${RECODE_AGENT_SYSTEMD_SERVICE_NAME}"
systemctl start "${RECODE_AGENT_SYSTEMD_SERVICE_NAME}"