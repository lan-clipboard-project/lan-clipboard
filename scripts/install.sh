#!/bin/sh
# Run on the Docker host from a new deployment directory.
set -eu
umask 077

release_base=https://github.com/lan-clipboard-project/lan-clipboard/releases
project_name=lan-clipboard
install_dir=$(pwd -P)
ui_accent=6
# Publish the Compose file last, after its configuration is in place.
install_files='.env .env.example compose.bind.yaml compose.yaml'

# The release ships this script as a single file, including its messages.
msg() {
    case "$installer_language:$1" in
        zh-CN:usage) format='用法  sh install.sh [--lang zh-CN|en]\n默认跟随终端语言。语言无法识别时使用英语。' ;;
        en:usage) format='Usage  sh install.sh [--lang zh-CN|en]\nLanguage follows the terminal locale by default, with English as the fallback.' ;;
        zh-CN:missing_language) format='请为 --lang 指定 zh-CN 或 en。' ;;
        en:missing_language) format='Pass zh-CN or en to --lang.' ;;
        zh-CN:unsupported_language) format='不支持语言 %s。请选择 zh-CN 或 en。' ;;
        en:unsupported_language) format='Unsupported language %s. Choose zh-CN or en.' ;;
        zh-CN:unknown_argument) format='不认识参数 %s。可用 --help 查看用法。' ;;
        en:unknown_argument) format='Unknown argument %s. Run with --help for usage.' ;;
        zh-CN:setup_title) format='LAN Clipboard 初始化向导' ;;
        en:setup_title) format='LAN Clipboard Setup Wizard' ;;
        zh-CN:installation_failed) format='安装失败。' ;;
        en:installation_failed) format='Installation failed. ' ;;
        zh-CN:docker_required) format='请先安装 Docker Engine、Docker Desktop 或群晖 Container Manager，再运行脚本。' ;;
        en:docker_required) format='Install Docker Engine, Docker Desktop, or Synology Container Manager before running the script.' ;;
        zh-CN:compose_required) format='需要 Docker Compose 插件。请确认 docker compose version 可以运行。' ;;
        en:compose_required) format='The Docker Compose plugin is required. Make sure docker compose version works.' ;;
        zh-CN:docker_unreachable) format='无法连接 Docker。请启动 Docker，并使用有权运行 Docker 的账号执行脚本。' ;;
        en:docker_unreachable) format='Cannot connect to Docker. Start Docker and run the script with an account that has Docker access.' ;;
        zh-CN:linux_containers_required) format='请将 Docker 切换到 Linux 容器模式。' ;;
        en:linux_containers_required) format='Switch Docker to Linux container mode.' ;;
        zh-CN:docker_host_unknown) format='无法确定当前 Docker 主机。' ;;
        en:docker_host_unknown) format='Could not determine the current Docker host.' ;;
        zh-CN:local_docker_required) format='请在 Docker 主机上运行脚本，以便使用该主机的局域网地址。' ;;
        en:local_docker_required) format='Run this script on the Docker host so it can use the host'\''s local network address.' ;;
        zh-CN:curl_required) format='请安装 curl，再运行脚本。' ;;
        en:curl_required) format='Install curl before running the script.' ;;
        zh-CN:directory_occupied) format='当前目录已有 %s。首次安装请选择新目录。已有实例请在原目录启动，或按部署指南更新。' ;;
        en:directory_occupied) format='The current directory already contains %s. Use a new directory for a first installation. To manage an existing instance, use its original directory or follow the deployment guide to update it.' ;;
        zh-CN:containers_check_failed) format='无法检查现有容器。' ;;
        en:containers_check_failed) format='Could not check for existing containers.' ;;
        zh-CN:volumes_check_failed) format='无法检查现有数据卷。' ;;
        en:volumes_check_failed) format='Could not check for existing volumes.' ;;
        zh-CN:project_exists) format='当前 Docker 主机已有 LAN Clipboard 容器或数据卷。请在原部署目录管理该实例。' ;;
        en:project_exists) format='This Docker host already has LAN Clipboard containers or volumes. Manage the instance from its original deployment directory.' ;;
        zh-CN:please_wait) format='请稍候…' ;;
        en:please_wait) format='Please wait…' ;;
        zh-CN:latest_release_failed) format='查找最新 Release 失败。请确认 Docker 主机可以访问 GitHub。' ;;
        en:latest_release_failed) format='Could not find the latest release. Make sure the Docker host can reach GitHub.' ;;
        zh-CN:stable_release_missing) format='GitHub 没有返回可用的稳定版 Release。' ;;
        en:stable_release_missing) format='GitHub did not return a usable stable release.' ;;
        zh-CN:downloading_assets) format='正在下载 LAN Clipboard %s 的部署文件…' ;;
        en:downloading_assets) format='Downloading deployment files for LAN Clipboard %s…' ;;
        zh-CN:assets_download_failed) format='下载部署文件失败。请检查网络或 Release 附件后重试。' ;;
        en:assets_download_failed) format='Could not download the deployment files. Check your connection and the release assets, then try again.' ;;
        zh-CN:asset_empty) format='下载的 %s 为空。请检查 Release 附件。' ;;
        en:asset_empty) format='The downloaded %s is empty. Check the release assets.' ;;
        zh-CN:gum_checksum_failed) format='Gum 下载文件校验失败。' ;;
        en:gum_checksum_failed) format='The downloaded Gum archive failed checksum verification.' ;;
        zh-CN:plain_prompts) format='终端界面暂不可用，继续使用文字提示。' ;;
        en:plain_prompts) format='The interactive terminal UI is unavailable. Continuing with text prompts.' ;;
        zh-CN:keyboard_help) format='↑/↓ 选择 · Enter 确认 · Esc 退出' ;;
        en:keyboard_help) format='↑/↓ Select · Enter Confirm · Esc Cancel' ;;
        zh-CN:invalid_option) format='请选择菜单中的选项。' ;;
        en:invalid_option) format='Choose an option from the menu.' ;;
        zh-CN:select_option) format='请选择' ;;
        en:select_option) format='Select an option' ;;
        zh-CN:invalid_option_number) format='请输入 1 或 2。' ;;
        en:invalid_option_number) format='Enter 1 or 2.' ;;
        zh-CN:access_question) format='您打算如何让 LAN Clipboard 提供服务？' ;;
        en:access_question) format='How would you like to access LAN Clipboard?' ;;
        zh-CN:lan_address_option) format='局域网（局域网中的其他设备可通过 %s 访问 LAN Clipboard 服务）' ;;
        en:lan_address_option) format='Local network (other devices on your network can connect via %s)' ;;
        zh-CN:lan_manual_option) format='局域网（手动填写本机的局域网 IPv4 地址）' ;;
        en:lan_manual_option) format='Local network (enter this host'\''s private IPv4 address)' ;;
        zh-CN:loopback_option) format='仅本机（仅允许通过 127.0.0.1 访问服务，适合通过同机反向代理使用域名）' ;;
        en:loopback_option) format='This device only (127.0.0.1, suitable for a reverse proxy on this host)' ;;
        zh-CN:loopback_summary) format='仅本机（127.0.0.1）' ;;
        en:loopback_summary) format='This device only (127.0.0.1)' ;;
        zh-CN:lan_summary) format='局域网（%s）' ;;
        en:lan_summary) format='Local network (%s)' ;;
        zh-CN:lan_label) format='局域网' ;;
        en:lan_label) format='Local network' ;;
        zh-CN:ipv4_question) format='您打算使用哪个局域网 IPv4 地址？' ;;
        en:ipv4_question) format='Which private IPv4 address would you like to use?' ;;
        zh-CN:invalid_ipv4) format='请输入私有 IPv4 地址，例如 192.168.1.10。' ;;
        en:invalid_ipv4) format='Enter a private IPv4 address, such as 192.168.1.10.' ;;
        zh-CN:invalid_number) format='请输入 %s 到 %s 之间的整数。' ;;
        en:invalid_number) format='Enter an integer between %s and %s.' ;;
        zh-CN:timezone_question) format='您希望 LAN Clipboard 使用哪个时区？' ;;
        en:timezone_question) format='Which time zone should LAN Clipboard use?' ;;
        zh-CN:timezone_search) format='搜索时区，例如 Tokyo 或 UTC' ;;
        en:timezone_search) format='Search time zones, such as Tokyo or UTC' ;;
        zh-CN:invalid_timezone) format='请选择列表中的时区。' ;;
        en:invalid_timezone) format='Choose a time zone from the list.' ;;
        zh-CN:timezone_show_all) format='查看全部时区' ;;
        en:timezone_show_all) format='Show all time zones' ;;
        zh-CN:timezone_number) format='请选择时区编号' ;;
        en:timezone_number) format='Choose a time zone number' ;;
        zh-CN:no_lan_address) format='未检测到局域网 IPv4，您可以选择仅本机访问，或手动填写地址。' ;;
        en:no_lan_address) format='No local IPv4 address was detected. You can limit access to this device or enter an address manually.' ;;
        zh-CN:setup_question) format='您打算如何配置 LAN Clipboard？' ;;
        en:setup_question) format='How would you like to configure LAN Clipboard?' ;;
        zh-CN:use_defaults) format='使用默认配置（推荐）' ;;
        en:use_defaults) format='Use defaults (recommended)' ;;
        zh-CN:custom_settings) format='自定义配置' ;;
        en:custom_settings) format='Customize settings' ;;
        zh-CN:port_question) format='您打算让 LAN Clipboard 使用哪个端口？' ;;
        en:port_question) format='Which port should LAN Clipboard use?' ;;
        zh-CN:retention_question) format='您希望未置顶的内容保留多少天？' ;;
        en:retention_question) format='How many days should unpinned items be kept?' ;;
        zh-CN:file_size_question) format='您希望单个上传文件最大为多少 MB？' ;;
        en:file_size_question) format='What should the upload limit per file be in MB?' ;;
        zh-CN:text_size_question) format='您希望单条文字最多占用多少字节？' ;;
        en:text_size_question) format='What should the size limit per text item be in bytes?' ;;
        zh-CN:thumbnail_question) format='您希望为超过多少 MB 的图片生成缩略图？' ;;
        en:thumbnail_question) format='Above what size in MB should images get thumbnails?' ;;
        zh-CN:preview_question) format='您希望启用 Word 和 PowerPoint 文件预览吗？' ;;
        en:preview_question) format='Would you like to enable Word and PowerPoint previews?' ;;
        zh-CN:preview_tip) format='启用后会额外拉取并运行 Gotenberg 容器。\n镜像占用磁盘空间，转换文件时会使用 CPU 和内存。' ;;
        en:preview_tip) format='Previews download and run an additional Gotenberg container.\nIts image uses disk space, and file conversion uses CPU and memory.' ;;
        zh-CN:enable) format='启用' ;;
        en:enable) format='Enable' ;;
        zh-CN:disable) format='不启用' ;;
        en:disable) format='Disable' ;;
        zh-CN:save_asset_failed) format='无法在当前目录保存 %s。' ;;
        en:save_asset_failed) format='Could not save %s in the current directory.' ;;
        zh-CN:start_failed) format='配置已保存。请检查上方 Docker 错误，解决后在当前目录运行 docker compose up -d。' ;;
        en:start_failed) format='The configuration was saved. Check the Docker error above, fix it, then run docker compose up -d in this directory.' ;;
        zh-CN:starting_containers) format='容器正在启动。可运行 docker compose ps 查看就绪状态。' ;;
        en:starting_containers) format='The containers are starting. Run docker compose ps to check their status.' ;;
        zh-CN:stage_failed) format='无法在当前目录创建临时目录。请检查目录的写入权限。' ;;
        en:stage_failed) format='Could not create a temporary directory here. Check that the current directory is writable.' ;;
        zh-CN:invalid_compose) format='生成的 Docker Compose 配置无效。' ;;
        en:invalid_compose) format='The generated Docker Compose configuration is invalid.' ;;
        zh-CN:start_question) format='立刻启动 LAN Clipboard？' ;;
        en:start_question) format='Start LAN Clipboard now?' ;;
        zh-CN:start_now) format='是，立刻启动' ;;
        en:start_now) format='Yes, start now' ;;
        zh-CN:start_later) format='暂不，稍后我手动启动' ;;
        en:start_later) format='Later, I will start it manually' ;;
        zh-CN:open_now) format='打开 LAN Clipboard' ;;
        en:open_now) format='Open LAN Clipboard' ;;
        zh-CN:manual_start) format='在当前目录运行以下命令启动 LAN Clipboard。' ;;
        en:manual_start) format='Run the following command in this directory to start LAN Clipboard.' ;;
        zh-CN:open_later) format='启动后打开  %s' ;;
        en:open_later) format='Once it is running, open  %s' ;;
        *) printf 'Unknown message key %s\n' "$1" >&2; return 1 ;;
    esac
    shift
    printf "$format" "$@"
}

say() {
    printf '%s\n' "$(msg "$@")"
}

select_language() {
    case "${LC_ALL:-${LC_MESSAGES:-${LANG:-}}}" in
        zh|zh[_@.-]*) installer_language=zh-CN ;;
        *) installer_language=en ;;
    esac

    show_usage=0
    while [ "$#" -gt 0 ]; do
        case "$1" in
            --lang)
                [ "$#" -ge 2 ] || fail missing_language
                requested_language=$2
                shift ;;
            --lang=*) requested_language=${1#--lang=} ;;
            -h|--help) show_usage=1; shift; continue ;;
            *) fail unknown_argument "$1" ;;
        esac
        case "$requested_language" in
            en|zh-CN) installer_language=$requested_language ;;
            *) fail unsupported_language "$requested_language" ;;
        esac
        shift
    done
    if [ "$show_usage" = 1 ]; then
        say usage
        exit 0
    fi
}

color_enabled() {
    [ -t "$1" ] && [ "${TERM:-dumb}" != dumb ] && [ -z "${NO_COLOR:-}" ]
}

show_banner() {
    printf '\n'
    if color_enabled 1; then
        # Preserve the exported banner's ANSI truecolor gradient.
        printf '%b\n' \
            ' \033[38;2;255;81;138m_      \033[38;2;255;96;127m_   \033[38;2;255;104;120m_  \033[38;2;255;110;115m_    \033[38;2;255;118;106m_\033[38;2;255;120;104m_\033[38;2;255;122;102m_ \033[38;2;255;125;97m_ \033[38;2;255;128;93m_      \033[38;2;255;139;76m_                      \033[38;2;255;192;101m_\033[0m' \
            '\033[38;2;255;84;136m| \033[38;2;255;88;133m|    \033[38;2;255;98;125m/\033[38;2;255;100;124m_\033[38;2;255;102;122m\\ \033[38;2;255;106;119m| \033[38;2;255;110;115m\\\033[38;2;255;112;113m| \033[38;2;255;115;109m|  \033[38;2;255;120;104m/ \033[38;2;255;123;100m_\033[38;2;255;125;97m_\033[38;2;255;126;95m| \033[38;2;255;130;90m(\033[38;2;255;131;88m_\033[38;2;255;133;85m)\033[38;2;255;134;83m_ \033[38;2;255;137;77m_\033[38;2;255;139;76m_\033[38;2;255;141;77m| \033[38;2;255;146;80m|\033[38;2;255;149;81m_\033[38;2;255;151;82m_  \033[38;2;255;158;85m_\033[38;2;255;160;86m_\033[38;2;255;163;87m_  \033[38;2;255;169;90m_\033[38;2;255;171;91m_ \033[38;2;255;176;93m_ \033[38;2;255;180;95m_ \033[38;2;255;185;97m_ \033[38;2;255;189;99m_\033[38;2;255;192;101m_\033[38;2;255;194;102m| \033[38;2;255;198;104m|\033[0m' \
            '\033[38;2;255;88;133m| \033[38;2;255;93;130m|\033[38;2;255;94;129m_\033[38;2;255;96;127m_ \033[38;2;255;100;124m/ \033[38;2;255;104;120m_ \033[38;2;255;108;117m\\\033[38;2;255;110;115m| \033[38;2;255;113;111m.\033[38;2;255;115;109m` \033[38;2;255;118;106m| \033[38;2;255;122;102m| \033[38;2;255;125;97m(\033[38;2;255;126;95m_\033[38;2;255;128;93m_\033[38;2;255;130;90m| \033[38;2;255;133;85m| \033[38;2;255;136;80m| \033[38;2;255;139;76m'\''\033[38;2;255;141;77m_ \033[38;2;255;146;80m\\ \033[38;2;255;151;82m'\''\033[38;2;255;153;83m_ \033[38;2;255;158;85m\\\033[38;2;255;160;86m/ \033[38;2;255;165;88m_ \033[38;2;255;169;90m\\\033[38;2;255;171;91m/ \033[38;2;255;176;93m_\033[38;2;255;178;94m` \033[38;2;255;183;96m| \033[38;2;255;187;98m'\''\033[38;2;255;189;99m_\033[38;2;255;192;101m/ \033[38;2;255;196;103m_\033[38;2;255;198;104m` \033[38;2;255;202;106m|\033[0m' \
            '\033[38;2;255;93;130m|\033[38;2;255;94;129m_\033[38;2;255;96;127m_\033[38;2;255;98;125m_\033[38;2;255;100;124m_\033[38;2;255;102;122m/\033[38;2;255;104;120m_\033[38;2;255;106;119m/ \033[38;2;255;110;115m\\\033[38;2;255;112;113m_\033[38;2;255;113;111m\\\033[38;2;255;115;109m_\033[38;2;255;117;107m|\033[38;2;255;118;106m\\\033[38;2;255;120;104m_\033[38;2;255;122;102m|  \033[38;2;255;126;95m\\\033[38;2;255;128;93m_\033[38;2;255;130;90m_\033[38;2;255;131;88m_\033[38;2;255;133;85m|\033[38;2;255;134;83m_\033[38;2;255;136;80m|\033[38;2;255;137;77m_\033[38;2;255;139;76m| \033[38;2;255;144;79m.\033[38;2;255;146;80m_\033[38;2;255;149;81m_\033[38;2;255;151;82m/\033[38;2;255;153;83m_\033[38;2;255;156;84m.\033[38;2;255;158;85m_\033[38;2;255;160;86m_\033[38;2;255;163;87m/\033[38;2;255;165;88m\\\033[38;2;255;167;89m_\033[38;2;255;169;90m_\033[38;2;255;171;91m_\033[38;2;255;174;92m/\033[38;2;255;176;93m\\\033[38;2;255;178;94m_\033[38;2;255;180;95m_\033[38;2;255;183;96m,\033[38;2;255;185;97m_\033[38;2;255;187;98m|\033[38;2;255;189;99m_\033[38;2;255;192;101m| \033[38;2;255;196;103m\\\033[38;2;255;198;104m_\033[38;2;255;200;105m_\033[38;2;255;202;106m,\033[38;2;255;204;107m_\033[38;2;255;207;108m|\033[0m' \
            '                           \033[38;2;255;144;79m|\033[38;2;255;146;80m_\033[38;2;255;149;81m|\033[0m'
        printf '\033[0m'
    else
        printf '%s\n' \
            ' _      _   _  _    ___ _ _      _                      _' \
            '| |    /_\ | \| |  / __| (_)_ __| |__  ___  __ _ _ _ __| |' \
            '| |__ / _ \| .` | | (__| | | '\''_ \ '\''_ \/ _ \/ _` | '\''_/ _` |' \
            '|____/_/ \_\_|\_|  \___|_|_| .__/_.__/\___/\__,_|_| \__,_|' \
            '                           |_|'
    fi
    printf '\n  🚀 %s\n\n' "$(msg setup_title)"
}

fail() {
    printf '%s%s\n' "$(msg installation_failed)" "$(msg "$@")" >&2
    exit 1
}

check_docker() {
    command -v docker >/dev/null 2>&1 ||
        fail docker_required
    docker compose version >/dev/null ||
        fail compose_required
    docker_os=$(docker info --format '{{.OSType}}') ||
        fail docker_unreachable
    [ "$docker_os" = linux ] || fail linux_containers_required

    endpoint=${DOCKER_HOST:-}
    if [ -n "${DOCKER_CONTEXT:-}" ] || [ -z "$endpoint" ]; then
        endpoint=$(docker context inspect --format '{{.Endpoints.docker.Host}}') ||
            fail docker_host_unknown
    fi
    case "$endpoint" in
        unix://*) ;;
        *) fail local_docker_required ;;
    esac
    command -v curl >/dev/null 2>&1 || fail curl_required
}

exists() {
    [ -e "$1" ] || [ -L "$1" ]
}

check_directory() {
    for file in $install_files compose.yml docker-compose.yml docker-compose.yaml \
        compose.override.yaml compose.override.yml docker-compose.override.yml docker-compose.override.yaml; do
        if exists "$install_dir/$file"; then
            fail directory_occupied "$file"
        fi
    done
}

check_project() {
    containers=$(docker ps --all --quiet --filter "label=com.docker.compose.project=$project_name") ||
        fail containers_check_failed
    volumes=$(docker volume ls --quiet --filter "name=^${project_name}_app-data$") ||
        fail volumes_check_failed
    if [ -n "$containers$volumes" ]; then
        fail project_exists
    fi
}

show_bootstrap_spinner() {
    spinner_message=$(msg please_wait)
    spinner_color=
    spinner_reset=
    if color_enabled 2; then
        spinner_color=$(printf '\033[3%sm' "$ui_accent")
        spinner_reset=$(printf '\033[0m')
    fi
    while :; do
        for frame in ⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏; do
            printf '\r%s%s%s  %s' "$spinner_color" "$frame" "$spinner_reset" "$spinner_message" >&2
            sleep 0.12 2>/dev/null || sleep 1
        done
    done
}

stop_bootstrap_spinner() {
    [ -n "$bootstrap_spinner_pid" ] || return 0
    kill "$bootstrap_spinner_pid" 2>/dev/null || :
    wait "$bootstrap_spinner_pid" 2>/dev/null || :
    bootstrap_spinner_pid=
    printf '\r\033[2K' >&2
}

cleanup() {
    stop_bootstrap_spinner
    if [ "$installed" = 0 ]; then
        for file in $install_files; do
            if [ ! -L "$install_dir/$file" ] && [ "$stage/$file" -ef "$install_dir/$file" ]; then
                rm -f "$install_dir/$file"
            fi
        done
    fi
    rm -rf "$stage"
}

fetch() {
    download_title=$1
    shift
    set -- curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' \
        --connect-timeout 10 --max-time 60 --retry 2 "$@"
    if [ -n "$prompt_ui" ] && [ -n "$download_title" ]; then
        set -- "$prompt_ui" spin --spinner dot --spinner.foreground "$ui_accent" \
            --title "$download_title" --title.foreground "$ui_accent" --show-error -- "$@"
    fi
    "$@"
}

download_release() {
    release_url=$(fetch '' --output /dev/null --write-out '%{url_effective}' "$release_base/latest") ||
        fail latest_release_failed
    tag=${release_url#"$release_base/tag/"}
    printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$' ||
        fail stable_release_missing

    release_assets=$release_base/download/$tag
    fetch "$(msg downloading_assets "$tag")" --fail-early \
        --output "$stage/compose.yaml" "$release_assets/compose.yaml" \
        --output "$stage/.env.example" "$release_assets/env.example" \
        --output "$stage/compose.bind.yaml" "$release_assets/compose.bind.yaml" ||
        fail assets_download_failed
    for asset in compose.yaml .env.example compose.bind.yaml; do
        [ -s "$stage/$asset" ] || fail asset_empty "$asset"
    done
}

prepare_prompt_ui() {
    gum_version=2.0.2
    # These hashes pin the archives published by Charm's v2.0.2 release.
    case "$(uname -s):$(uname -m)" in
        Darwin:arm64)
            gum_platform=Darwin_arm64
            gum_checksum=4777a69b1170b8db23c95d5889fb32186cfda1a3ac950d339aa17e3513633890 ;;
        Darwin:x86_64)
            gum_platform=Darwin_x86_64
            gum_checksum=5374966c7c7199ea879fcaa525ddc6d447a098d3d35496e430a9a1ef38d30485 ;;
        Linux:aarch64|Linux:arm64)
            gum_platform=Linux_arm64
            gum_checksum=8ebf8b54ec1e8c81f2bb58b59ff9b70998186a4d11375f0cf357b80e0ccfa1d5 ;;
        Linux:x86_64)
            gum_platform=Linux_x86_64
            gum_checksum=d842e06d93dbed90af48cb8dd10698db6f22e331fc40346bb37bbc753109edc2 ;;
        *) return 1 ;;
    esac
    command -v tar >/dev/null 2>&1 || return 1
    if command -v sha256sum >/dev/null 2>&1; then
        set -- sha256sum
    elif command -v shasum >/dev/null 2>&1; then
        set -- shasum -a 256
    else
        return 1
    fi

    gum_archive="gum_${gum_version}_${gum_platform}"
    fetch '' --max-time 30 --retry 0 --output "$stage/gum.tar.gz" \
        "https://github.com/charmbracelet/gum/releases/download/v$gum_version/$gum_archive.tar.gz" || return 1
    checksum=$("$@" < "$stage/gum.tar.gz") || return 1
    if [ "${checksum%% *}" != "$gum_checksum" ]; then
        say gum_checksum_failed >&2
        return 1
    fi

    tar -xOzf "$stage/gum.tar.gz" "$gum_archive/gum" > "$stage/gum" || return 1
    chmod +x "$stage/gum" && "$stage/gum" --version >/dev/null 2>&1
}

initialize_prompt_ui() {
    # Use a terminal spinner while the executable that provides gum spin is unavailable.
    show_bootstrap_spinner &
    bootstrap_spinner_pid=$!
    # Keep the parent terminal available for cleanup even if setup is interrupted.
    if (prepare_prompt_ui) > "$stage/gum-setup.log" 2>&1; then
        prompt_ui=$stage/gum
    fi
    stop_bootstrap_spinner
    if [ -z "$prompt_ui" ]; then
        cat "$stage/gum-setup.log" >&2
        say plain_prompts >&2
    fi
}

interface_addresses() {
    if command -v ip >/dev/null 2>&1 &&
        addresses=$(ip -o -4 addr show up scope global 2>/dev/null) && [ -n "$addresses" ]; then
        printf '%s\n' "$addresses" | awk '$3 == "inet" { sub(/@.*/, "", $2); sub(/\/.*/, "", $4); print $2, $4 }'
    elif command -v ifconfig >/dev/null 2>&1 &&
        addresses=$(ifconfig 2>/dev/null); then
        printf '%s\n' "$addresses" | awk '
            /^[^[:space:]]/ { iface = $1; sub(/:$/, "", iface) }
            $1 == "inet" { sub(/^addr:/, "", $2); print iface, $2 }
        '
    fi
}

default_interface() {
    case "$(uname -s)" in
        Darwin)
            route -n get default 2>/dev/null | awk '$1 == "interface:" { print $2; exit }'
            ;;
        Linux)
            if command -v ip >/dev/null 2>&1 && routes=$(ip -4 route show default 2>/dev/null); then
                printf '%s\n' "$routes" | awk '$1 == "default" {
                    for (i = 2; i < NF; i++) if ($i == "dev") { print $(i+1); exit }
                }'
            elif command -v route >/dev/null 2>&1; then
                route -n 2>/dev/null | awk '$1 == "0.0.0.0" && $4 ~ /G/ { print $NF; exit }'
            fi
            ;;
    esac
}

private_ipv4() {
    printf '%s\n' "$1" | awk -F. '
        NF == 4 {
            for (i = 1; i <= 4; i++)
                if ($i !~ /^(0|[1-9][0-9]?[0-9]?)$/ || $i > 255) exit 1
            private = ($1 == 10 || ($1 == 172 && $2 >= 16 && $2 <= 31) || ($1 == 192 && $2 == 168))
        }
        END { exit !private }
    '
}

find_lan_address() {
    preferred=$(default_interface)
    selected=
    while read -r iface address; do
        case "$iface" in
            docker*|br-*|virbr*|veth*|tun*|tap*|utun*|wg*|tailscale*) continue ;;
        esac
        private_ipv4 "$address" || continue
        if [ "$iface" = "$preferred" ]; then
            printf '%s\n' "$address"
            return
        fi
        selected=${selected:-$address}
    done <<EOF
$(interface_addresses)
EOF
    printf '%s\n' "$selected"
}

run_prompt() {
    if answer=$("$prompt_ui" "$@"); then
        return
    else
        prompt_status=$?
    fi
    if [ "$prompt_status" -eq 1 ] && [ -t 2 ]; then
        # Gum prints one line when the user leaves a prompt without submitting it.
        printf '\033[1F\033[2K' >&2
    fi
    exit "$prompt_status"
}

print_styled() {
    message=$1
    shift
    if [ -n "$prompt_ui" ] && "$prompt_ui" style "$@" "$message"; then
        return
    fi
    printf '%s\n' "$message"
}

record_answer() {
    if color_enabled 2; then
        printf '  %s  \033[1;3%sm%s\033[0m\n' "$1" "$ui_accent" "$2" >&2
    else
        printf '  %s  %s\n' "$1" "$2" >&2
    fi
}

prompt() {
    if [ -n "$prompt_ui" ]; then
        run_prompt input --no-show-help --header "$1" --header.bold --header.foreground "$ui_accent" \
            --placeholder "$2" --prompt '› ' --prompt.foreground "$ui_accent" \
            --cursor.foreground "$ui_accent"
    else
        printf '%s ' "$1" >&2
        if [ -n "$2" ]; then
            printf '[%s] ' "$2" >&2
        fi
        IFS= read -r answer || exit 130
        [ -t 0 ] || printf '\n' >&2
    fi
    answer=${answer:-$2}
}

# Each option is a translated label followed by its fixed value, separated by |.
choose_option() {
    question_header=$1
    if [ -n "${5:-}" ]; then
        if color_enabled 2; then
            question_header=$(printf '%s\n\033[22;90m%s\033[0m' "$1" "$5")
        else
            question_header=$(printf '%s\n%s' "$1" "$5")
        fi
    fi

    if [ -n "$prompt_ui" ]; then
        if [ "$2" = 1 ]; then
            set -- "$3" "$4"
        else
            set -- "$4" "$3"
        fi
        run_prompt choose --no-show-help --label-delimiter '|' --header "$question_header" \
            --header.bold --header.foreground "$ui_accent" \
            --cursor '› ' --cursor.foreground "$ui_accent" --selected.foreground "$ui_accent" "$@"
        [ "$answer" = "${1#*|}" ] || [ "$answer" = "${2#*|}" ] || fail invalid_option
        return
    fi

    printf '\n%s\n  1  %s\n  2  %s\n' "$question_header" "${3%%|*}" "${4%%|*}" >&2
    while :; do
        prompt "$(msg select_option)" "$2"
        case "$answer" in
            1|[yY]|[yY][eE][sS]) answer=${3#*|}; return ;;
            2|[nN]|[nN][oO]) answer=${4#*|}; return ;;
        esac
        say invalid_option_number >&2
    done
}

confirm() {
    choose_option "$@"
    if [ "$answer" = "${3#*|}" ]; then
        record_answer "$1" "${3%%|*}"
        return 0
    fi
    record_answer "$1" "${4%%|*}"
    return 1
}

prompt_address() {
    access_question=$(msg access_question)
    if [ -n "$lan_address" ]; then
        lan_option=$(msg lan_address_option "$lan_address")
        default_access=1
    else
        lan_option=$(msg lan_manual_option)
        default_access=2
    fi
    choose_option "$access_question" "$default_access" "$lan_option|lan" "$(msg loopback_option)|loopback"
    if [ "$answer" = loopback ]; then
        bind_address=127.0.0.1
        record_answer "$access_question" "$(msg loopback_summary)"
        return
    fi
    if [ -n "$lan_address" ]; then
        bind_address=$lan_address
        record_answer "$access_question" "$(msg lan_summary "$bind_address")"
        return
    fi

    record_answer "$access_question" "$(msg lan_label)"
    address_question=$(msg ipv4_question)
    while :; do
        prompt "$address_question" ''
        if private_ipv4 "$answer"; then
            bind_address=$answer
            record_answer "$address_question" "$bind_address"
            return
        fi
        say invalid_ipv4 >&2
    done
}

read_number() {
    while :; do
        prompt "$1" "$2"
        case "$answer" in
            *[!0-9]*) ;;
            *)
                if [ "${#answer}" -le 10 ] && [ "$answer" -ge "$3" ] && [ "$answer" -le "$4" ]; then
                    return
                fi
                ;;
        esac
        say invalid_number "$3" "$4" >&2
    done
}

prompt_number() {
    read_number "$@"
    record_answer "$1" "$answer"
}

select_timezone() {
    timezone_question=$(msg timezone_question)
    zones_file=$stage/timezones
    {
        printf '%s\n' "$timezone" UTC Asia/Shanghai Asia/Hong_Kong Asia/Taipei Asia/Singapore \
            Asia/Tokyo Europe/London Europe/Paris America/New_York America/Los_Angeles Australia/Sydney
        if [ -r /usr/share/zoneinfo/zone.tab ]; then
            awk '!/^#/ && NF >= 3 { print $3 }' /usr/share/zoneinfo/zone.tab
        fi
    } | awk 'NF && !seen[$0]++' > "$zones_file"

    if [ -n "$prompt_ui" ]; then
        run_prompt filter --no-show-help --strict --height 12 --header "$timezone_question" --header.bold \
            --header.foreground "$ui_accent" --indicator.foreground "$ui_accent" \
            --match.foreground "$ui_accent" \
            --placeholder "$(msg timezone_search)" < "$zones_file"
        grep -Fxq -e "$answer" "$zones_file" || fail invalid_timezone
        timezone=$answer
    else
        printf '\n%s\n' "$timezone_question" >&2
        zone_count=$(awk 'END { print NR }' "$zones_file")
        visible_zones=$zone_count
        [ "$visible_zones" -le 12 ] || visible_zones=12
        while :; do
            awk -v limit="$visible_zones" 'NR <= limit { printf "  %3d. %s\n", NR, $0 }' "$zones_file" >&2
            first_choice=1
            if [ "$visible_zones" -lt "$zone_count" ]; then
                printf '    0. %s\n' "$(msg timezone_show_all)" >&2
                first_choice=0
            fi
            read_number "$(msg timezone_number)" 1 "$first_choice" "$visible_zones"
            [ "$answer" -eq 0 ] || break
            visible_zones=$zone_count
        done
        timezone=$(awk -v choice="$answer" 'NR == choice { print; exit }' "$zones_file")
    fi
    record_answer "$timezone_question" "$timezone"
}

template_value() {
    value=$(awk -v key="$1" 'index($0, key "=") == 1 { print substr($0, length(key) + 2); exit }' "$stage/.env.example")
    printf '%s\n' "${value:-$2}"
}

configure() {
    lan_address=$(find_lan_address)
    bind_address=${lan_address:-127.0.0.1}

    host_port=$(template_value CLIP_HOST_PORT 18080)
    timezone=$(template_value TZ Asia/Shanghai)
    retention_days=$(template_value CLIP_RETENTION_DAYS 30)
    max_file_size=$(template_value CLIP_MAX_FILE_SIZE 300)
    max_text_size=$(template_value CLIP_MAX_TEXT_SIZE 307200)
    thumbnail_threshold=$(template_value CLIP_THUMBNAIL_THRESHOLD 10)
    office_profile=
    converter_url=

    if [ -z "$lan_address" ]; then
        say no_lan_address
    fi
    if confirm "$(msg setup_question)" 1 "$(msg use_defaults)|default" "$(msg custom_settings)|custom"; then
        [ -n "$lan_address" ] || prompt_address
        return
    fi

    prompt_address
    prompt_number "$(msg port_question)" "$host_port" 1 65535
    host_port=$answer
    select_timezone
    prompt_number "$(msg retention_question)" "$retention_days" 1 2147483647
    retention_days=$answer
    prompt_number "$(msg file_size_question)" "$max_file_size" 1 2147483647
    max_file_size=$answer
    prompt_number "$(msg text_size_question)" "$max_text_size" 1 2147483647
    max_text_size=$answer
    prompt_number "$(msg thumbnail_question)" "$thumbnail_threshold" 0 2147483647
    thumbnail_threshold=$answer
    if confirm "$(msg preview_question)" 2 "$(msg enable)|enable" "$(msg disable)|disable" \
        "$(msg preview_tip)"; then
        office_profile=office
        converter_url=http://office:3000
    fi
}

write_env() {
    printf '%s=%s\n' \
        CLIP_BIND_ADDRESS "$bind_address" CLIP_HOST_PORT "$host_port" TZ "$timezone" \
        CLIP_RETENTION_DAYS "$retention_days" CLIP_MAX_FILE_SIZE "$max_file_size" \
        CLIP_MAX_TEXT_SIZE "$max_text_size" CLIP_THUMBNAIL_THRESHOLD "$thumbnail_threshold" \
        COMPOSE_PROFILES "$office_profile" CLIP_OFFICE_CONVERTER_URL "$converter_url" > "$stage/overrides"
    awk '
        FNR == NR {
            key = $0
            sub(/=.*/, "", key)
            setting[key] = $0
            order[++count] = key
            next
        }
        {
            key = $0
            sub(/^[[:space:]]*(#[[:space:]]*)?/, "", key)
            sub(/=.*/, "", key)
            if (key in setting) {
                if (!written[key]++) print setting[key]
                next
            }
            print
        }
        END {
            for (i = 1; i <= count; i++)
                if (!written[order[i]]) print setting[order[i]]
        }
    ' "$stage/overrides" "$stage/.env.example" > "$stage/.env"
}

compose() {
    docker compose --project-name "$project_name" --project-directory "$install_dir" \
        --env-file "$compose_dir/.env" -f "$compose_dir/compose.yaml" "$@"
}

publish_files() {
    check_directory
    for file in $install_files; do
        # A hard link publishes a complete file and fails if another install created it.
        ln "$stage/$file" "$install_dir/$file" || fail save_asset_failed "$file"
    done
    installed=1
    compose_dir=$install_dir
}

start() {
    check_project
    # Older Compose plugins can start the project without waiting for its health check.
    case "$(compose up --help)" in
        *--wait-timeout*) set -- --wait --wait-timeout 90 ;;
        *) set -- ;;
    esac
    if ! compose up -d --no-build "$@"; then
        fail start_failed
    fi
    if [ "$#" -eq 0 ]; then
        say starting_containers
    fi
}

select_language "$@"
show_banner
check_docker
check_directory
check_project

# Setup choices take precedence over Compose settings exported by the caller.
unset CLIP_IMAGE CLIP_BIND_ADDRESS CLIP_HOST_PORT CLIP_PUBLIC_ORIGIN CLIP_MAX_FILE_SIZE \
    CLIP_MAX_TEXT_SIZE CLIP_RETENTION_DAYS CLIP_THUMBNAIL_THRESHOLD CLIP_OFFICE_CONVERTER_URL \
    COMPOSE_PROFILES COMPOSE_REMOVE_ORPHANS TZ

stage=$(mktemp -d "$install_dir/.lan-clipboard-install.XXXXXXXX") ||
    fail stage_failed
installed=0
prompt_ui=
bootstrap_spinner_pid=
trap cleanup 0
trap 'exit 130' INT
trap 'exit 143' TERM
compose_dir=$stage

if [ -t 0 ] && [ -t 2 ] && [ "${TERM:-dumb}" != dumb ]; then
    initialize_prompt_ui
fi
download_release
if [ -n "$prompt_ui" ]; then
    say keyboard_help >&2
fi
configure
write_env
compose config --quiet || fail invalid_compose

web_url=http://$bind_address:$host_port
start_now=0
if confirm "$(msg start_question)" 1 "$(msg start_now)|start_now" "$(msg start_later)|start_later"; then
    start_now=1
fi

publish_files
if [ "$start_now" = 1 ]; then
    start
    printf '\n%s\n' "$(msg open_now)"
    print_styled "$web_url" --foreground "$ui_accent" --bold
else
    printf '\n%s\n\n' "$(msg manual_start)"
    print_styled 'docker compose up -d' --foreground "$ui_accent" --bold \
        --border rounded --border-foreground "$ui_accent" --padding '0 1'
    printf '\n%s\n' "$(msg open_later "$web_url")"
fi
