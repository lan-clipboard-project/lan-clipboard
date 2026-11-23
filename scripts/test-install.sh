#!/usr/bin/env bash
# Exercise the installer using release and Docker fixtures in a temporary directory.
set -euo pipefail
unset LC_ALL LC_MESSAGES
export LANG=zh_CN.UTF-8

export INSTALL_TEST_REPO
INSTALL_TEST_REPO="$(cd "$(dirname "$0")/.." && pwd)"
export INSTALL_TEST_DOCKER
INSTALL_TEST_DOCKER="$(command -v docker || true)"
export INSTALL_TEST_LN
INSTALL_TEST_LN="$(command -v ln)"
test_dir="$(mktemp -d)"
trap 'rm -rf "$test_dir"' EXIT
mock_bin="$test_dir/bin"
mkdir "$mock_bin"

cat > "$mock_bin/curl" <<'EOF'
#!/bin/sh
set -eu
base=https://github.com/lan-clipboard-project/lan-clipboard/releases
while [ "$#" -gt 0 ]; do
    case "$1" in
        --output) output=$2; shift ;;
        "$base/latest")
            printf '%s\n' "$1" >> requests
            printf '%s/tag/v1.2.3' "$base" ;;
        "$base/download/v1.2.3/compose.yaml")
            printf '%s\n' "$1" >> requests
            sed 's|lan-clipboard:local|ghcr.io/lan-clipboard-project/lan-clipboard:v1.2.3|' \
                "$INSTALL_TEST_REPO/compose.yaml" > "$output" ;;
        "$base/download/v1.2.3/env.example")
            printf '%s\n' "$1" >> requests
            case "${INSTALL_TEST_CASE:-}" in
                download-failure)
                    printf 'incomplete download\n' > "$output"
                    exit 22 ;;
                download-terminated)
                    printf 'incomplete download\n' > "$output"
                    kill -TERM "$PPID"
                    exit 22 ;;
            esac
            cp "$INSTALL_TEST_REPO/.env.example" "$output" ;;
        "$base/download/v1.2.3/compose.bind.yaml")
            printf '%s\n' "$1" >> requests
            cp "$INSTALL_TEST_REPO/compose.bind.yaml" "$output" ;;
        https://*) exit 22 ;;
    esac
    shift
done
EOF

cat > "$mock_bin/ln" <<'EOF'
#!/bin/sh
case "${INSTALL_TEST_CASE:-}:$2" in
    publish-failure:*/compose.bind.yaml) exit 1 ;;
esac
exec "$INSTALL_TEST_LN" "$@"
EOF

cat > "$mock_bin/docker" <<'EOF'
#!/bin/sh
set -eu
case "$1" in
    info)
        [ "${INSTALL_TEST_CASE:-}" != daemon-offline ] || exit 1
        printf 'linux\n' ;;
    context) printf 'unix:///var/run/docker.sock\n' ;;
    ps) ;;
    volume)
        if [ "${INSTALL_TEST_CASE:-}" = occupied ]; then
            printf 'lan-clipboard_app-data\n'
        fi ;;
    compose)
        if [ "$2" = version ]; then
            [ "${INSTALL_TEST_CASE:-}" != no-compose ]
            exit
        fi
        case "${10}" in
            config)
                [ -s "$7" ] && [ -s "$9" ]
                [ -z "${CLIP_IMAGE:-}${CLIP_BIND_ADDRESS:-}${COMPOSE_PROFILES:-}" ]
                if [ -n "$INSTALL_TEST_DOCKER" ]; then
                    exec "$INSTALL_TEST_DOCKER" "$@"
                fi ;;
            up)
                if [ "${11}" = --help ]; then
                    printf '%s\n' '--wait-timeout'
                else
                    [ "${INSTALL_TEST_CASE:-}" != start-failure ] || exit 1
                    [ "${11} ${12} ${13} ${14} ${15}" = '-d --no-build --wait --wait-timeout 90' ]
                    touch started
                fi ;;
            *) exit 1 ;;
        esac ;;
    *) exit 1 ;;
esac
EOF

cat > "$mock_bin/uname" <<'EOF'
#!/bin/sh
case "${INSTALL_TEST_CASE:-}" in
    mac) printf 'Darwin\n' ;;
    *) printf 'Linux\n' ;;
esac
EOF

cat > "$mock_bin/ip" <<'EOF'
#!/bin/sh
case "${INSTALL_TEST_CASE:-}" in
    mac|no-lan) exit 1 ;;
esac
case "$*" in
    '-o -4 addr show up scope global')
        printf '%s\n' '2: docker0 inet 172.17.0.1/16' \
            '3: eth0 inet 10.3.0.10/24' '4: eth1 inet 192.168.20.12/24' ;;
    '-4 route show default') printf 'default via 192.168.20.1 dev eth1\n' ;;
    *) exit 1 ;;
esac
EOF

cat > "$mock_bin/ifconfig" <<'EOF'
#!/bin/sh
if [ "${INSTALL_TEST_CASE:-}" = mac ]; then
    printf '%s\n' 'utun2: flags' '    inet 10.9.0.2 netmask 0xffffffff' \
        'en0: flags' '    inet 192.168.0.30 netmask 0xffffff00'
fi
EOF

cat > "$mock_bin/route" <<'EOF'
#!/bin/sh
printf '  interface: utun2\n'
EOF

chmod +x "$mock_bin/"*

run() (
    scenario=$1
    replies=$2
    shift 2
    cd "$test_dir/$scenario"
    unset DOCKER_HOST DOCKER_CONTEXT
    printf '%b' "$replies" | env PATH="$mock_bin:$PATH" INSTALL_TEST_CASE="$scenario" \
        CLIP_BIND_ADDRESS=0.0.0.0 CLIP_IMAGE=ignored COMPOSE_PROFILES=office \
        sh "$INSTALL_TEST_REPO/scripts/install.sh" "$@" > output 2>&1
)

expect_failure() {
    if run "$@"; then
        echo "Expected failure in $1" >&2
        return 1
    fi
}

expect_no_install() {
    for file in .env .env.example compose.bind.yaml compose.yaml; do
        [[ ! -e "$test_dir/$1/$file" && ! -L "$test_dir/$1/$file" ]]
    done
    ! compgen -G "$test_dir/$1/.lan-clipboard-install.*" > /dev/null
}

missing="$test_dir/no-docker"
mkdir "$missing"
run_without_docker() (
    cd "$missing"
    PATH="$missing" /bin/sh "$INSTALL_TEST_REPO/scripts/install.sh" "$@"
)
if run_without_docker > "$missing/output" 2>&1; then
    echo 'Installer succeeded without Docker' >&2
    exit 1
fi
grep -q '请先安装 Docker' "$missing/output"
grep -Fq '🚀 LAN Clipboard 初始化向导' "$missing/output"
grep -Fxq ' _      _   _  _    ___ _ _      _                      _' "$missing/output"

LC_ALL=en_US.UTF-8 LC_MESSAGES=zh_CN.UTF-8 LANG=zh_CN.UTF-8 \
    run_without_docker --help > "$missing/help-en"
grep -Fq 'Usage  sh install.sh' "$missing/help-en"
LC_ALL= LC_MESSAGES=zh_CN.UTF-8 LANG=en_US.UTF-8 \
    run_without_docker --help > "$missing/help-zh"
grep -Fq '用法  sh install.sh' "$missing/help-zh"
LC_ALL=zh_CN.UTF-8 run_without_docker --lang en --help > "$missing/help-en"
grep -Fq 'Usage  sh install.sh' "$missing/help-en"
LC_ALL=en_US.UTF-8 run_without_docker --help --lang=zh-CN > "$missing/help-zh"
grep -Fq '用法  sh install.sh' "$missing/help-zh"
if LC_ALL=C run_without_docker > "$missing/preflight-en" 2>&1; then
    echo 'Installer succeeded without Docker' >&2
    exit 1
fi
grep -Fq '🚀 LAN Clipboard Setup Wizard' "$missing/preflight-en"
grep -Fq 'Installation failed. Install Docker Engine' "$missing/preflight-en"
for terminal_locale in '' fr_FR.UTF-8; do
    LANG="$terminal_locale" run_without_docker --help > "$missing/help-en"
    grep -Fq 'with English as the fallback.' "$missing/help-en"
done
if LC_ALL=en_US.UTF-8 run_without_docker --lang=fr > "$missing/invalid-lang" 2>&1; then
    echo 'Installer accepted an unsupported language' >&2
    exit 1
fi
grep -Fq 'Unsupported language fr.' "$missing/invalid-lang"
if run_without_docker --lang > "$missing/missing-lang" 2>&1; then
    echo 'Installer accepted a missing language' >&2
    exit 1
fi
grep -Fq '请为 --lang 指定 zh-CN 或 en。' "$missing/missing-lang"

for scenario in default custom english loopback mac no-lan occupied existing interrupted download-failure \
    download-terminated no-compose daemon-offline publish-failure start-failure; do
    mkdir "$test_dir/$scenario"
done

run default '\n\n'
grep -Fxq 'CLIP_BIND_ADDRESS=192.168.20.12' "$test_dir/default/.env"
grep -Fxq 'TZ=Asia/Shanghai' "$test_dir/default/.env"
grep -Fq 'http://192.168.20.12:18080' "$test_dir/default/output"
grep -Fxq '  您打算如何配置 LAN Clipboard？  使用默认配置（推荐）' "$test_dir/default/output"
grep -Fxq '  立刻启动 LAN Clipboard？  是，立刻启动' "$test_dir/default/output"
grep -Fq '暂不，稍后我手动启动' "$test_dir/default/output"
[[ -s "$test_dir/default/.env.example" && -s "$test_dir/default/compose.bind.yaml" && -f "$test_dir/default/started" ]]
mkdir "$test_dir/saved"
for file in .env .env.example compose.yaml compose.bind.yaml requests; do
    cp "$test_dir/default/$file" "$test_dir/saved/"
done
expect_failure default '\n\n'
grep -Fq '当前目录已有' "$test_dir/default/output"
for file in .env .env.example compose.yaml compose.bind.yaml requests; do
    cmp "$test_dir/default/$file" "$test_dir/saved/$file"
done

custom_replies='n\n1\n65536\n18081\n9999\n2\n45\n20\n1024\n0\ny\nn\n'
run custom "$custom_replies"
grep -Eq '^[[:space:]]*2\. UTC' "$test_dir/custom/output"
for setting in CLIP_BIND_ADDRESS=192.168.20.12 CLIP_HOST_PORT=18081 TZ=UTC CLIP_RETENTION_DAYS=45 \
    CLIP_MAX_FILE_SIZE=20 CLIP_MAX_TEXT_SIZE=1024 CLIP_THUMBNAIL_THRESHOLD=0 \
    COMPOSE_PROFILES=office CLIP_OFFICE_CONVERTER_URL=http://office:3000; do
    grep -Fxq "$setting" "$test_dir/custom/.env"
done
[[ ! -e "$test_dir/custom/started" ]]
grep -Fq 'http://192.168.20.12:18081' "$test_dir/custom/output"
grep -Fq '局域网中的其他设备可通过 192.168.20.12 访问 LAN Clipboard 服务' "$test_dir/custom/output"
grep -Fxq '  您打算如何让 LAN Clipboard 提供服务？  局域网（192.168.20.12）' "$test_dir/custom/output"
grep -Fxq '  您打算让 LAN Clipboard 使用哪个端口？  18081' "$test_dir/custom/output"
grep -Fxq '  您希望 LAN Clipboard 使用哪个时区？  UTC' "$test_dir/custom/output"
grep -Fxq '  您希望未置顶的内容保留多少天？  45' "$test_dir/custom/output"
grep -Fq 'Gotenberg 容器' "$test_dir/custom/output"
grep -Fxq '  您希望启用 Word 和 PowerPoint 文件预览吗？  启用' "$test_dir/custom/output"
grep -Fxq '  2  不启用' "$test_dir/custom/output"
grep -Fxq '  立刻启动 LAN Clipboard？  暂不，稍后我手动启动' "$test_dir/custom/output"
grep -Fxq 'docker compose up -d' "$test_dir/custom/output"
if grep -Eq '默认访问地址为|请在可信局域网使用|安装文件已保存' \
    "$test_dir/default/output" "$test_dir/custom/output"; then
    echo 'Installer displayed outdated copy' >&2
    exit 1
fi

LC_ALL= LC_MESSAGES= LANG=en_US.UTF-8 run english "$custom_replies"
cmp "$test_dir/custom/.env" "$test_dir/english/.env"
grep -Fq '🚀 LAN Clipboard Setup Wizard' "$test_dir/english/output"
grep -Fxq '  How would you like to configure LAN Clipboard?  Customize settings' "$test_dir/english/output"
grep -Fxq '  How would you like to access LAN Clipboard?  Local network (192.168.20.12)' "$test_dir/english/output"
grep -Fxq 'Enter an integer between 1 and 65535.' "$test_dir/english/output"
grep -Fxq '  Which time zone should LAN Clipboard use?  UTC' "$test_dir/english/output"
grep -Fxq '  Would you like to enable Word and PowerPoint previews?  Enable' "$test_dir/english/output"
grep -Fxq '  Start LAN Clipboard now?  Later, I will start it manually' "$test_dir/english/output"
grep -Fxq 'Run the following command in this directory to start LAN Clipboard.' "$test_dir/english/output"

run loopback '2\n2\n\n\n\n\n\n\n\n2\n'
grep -Fxq 'CLIP_BIND_ADDRESS=127.0.0.1' "$test_dir/loopback/.env"
grep -Fxq '  您打算如何让 LAN Clipboard 提供服务？  仅本机（127.0.0.1）' "$test_dir/loopback/output"
grep -Fxq '  您希望启用 Word 和 PowerPoint 文件预览吗？  不启用' "$test_dir/loopback/output"

run mac '\nn\n'
grep -Fxq 'CLIP_BIND_ADDRESS=192.168.0.30' "$test_dir/mac/.env"
run no-lan '\n1\n0.0.0.0\n192.168.1.17\nn\n'
grep -Fxq 'CLIP_BIND_ADDRESS=192.168.1.17' "$test_dir/no-lan/.env"
grep -Fxq '  您打算使用哪个局域网 IPv4 地址？  192.168.1.17' "$test_dir/no-lan/output"

for scenario in occupied no-compose daemon-offline; do
    expect_failure "$scenario" '\n\n'
    expect_no_install "$scenario"
    [[ ! -e "$test_dir/$scenario/requests" ]]
done
printf 'CLIP_HOST_PORT=12345\n' > "$test_dir/existing/.env"
expect_failure existing '\n\n'
grep -Fxq 'CLIP_HOST_PORT=12345' "$test_dir/existing/.env"
[[ ! -e "$test_dir/existing/requests" ]]

expect_failure download-failure ''
expect_no_install download-failure
status=0
run interrupted '' || status=$?
[[ "$status" -eq 130 ]]
expect_no_install interrupted
if grep -Eq '安装失败|输入已结束|交互提问已中断' "$test_dir/interrupted/output"; then
    echo 'Installer displayed an error on cancellation' >&2
    exit 1
fi
status=0
run download-terminated '' || status=$?
[[ "$status" -eq 143 ]]
expect_no_install download-terminated
expect_failure publish-failure '\nn\n'
expect_no_install publish-failure
grep -Fq '无法在当前目录保存 compose.bind.yaml' "$test_dir/publish-failure/output"

expect_failure start-failure '\n\n'
[[ -s "$test_dir/start-failure/compose.yaml" && -s "$test_dir/start-failure/.env" ]]
grep -Fq '配置已保存。请检查上方 Docker 错误，解决后在当前目录运行 docker compose up -d。' "$test_dir/start-failure/output"
if [[ -n "$INSTALL_TEST_DOCKER" ]]; then
    (cd "$test_dir/start-failure" && "$INSTALL_TEST_DOCKER" compose config --quiet)
fi

echo 'PASS installer: language selection, banner, defaults, time zones, LAN detection, preflight, and recovery'
