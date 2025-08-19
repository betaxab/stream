#!/usr/bin/env bash
echo=echo
for cmd in echo /bin/echo; do
    $cmd >/dev/null 2>&1 || continue

    if ! $cmd -e "" | grep -qE '^-e'; then
        echo=$cmd
        break
    fi
done

CSI=$($echo -e "\033[")
CEND="${CSI}0m"
CDGREEN="${CSI}32m"
CRED="${CSI}1;31m"
CGREEN="${CSI}1;32m"
CYELLOW="${CSI}1;33m"
CBLUE="${CSI}1;34m"
CMAGENTA="${CSI}1;35m"
CCYAN="${CSI}1;36m"

OUT_ALERT() {
    echo -e "${CYELLOW}$1${CEND}"
}

OUT_ERROR() {
    echo -e "${CRED}$1${CEND}"

    exit $?
}

OUT_INFO() {
    echo -e "${CCYAN}$1${CEND}"
}

if [[ -f /etc/redhat-release ]]; then
    release="centos"
elif cat /etc/issue | grep -q -E -i "debian"; then
    release="debian"
elif cat /etc/issue | grep -q -E -i "ubuntu"; then
    release="ubuntu"
elif cat /etc/issue | grep -q -E -i "centos|red hat|redhat"; then
    release="centos"
elif cat /proc/version | grep -q -E -i "raspbian|debian"; then
    release="debian"
elif cat /proc/version | grep -q -E -i "ubuntu"; then
    release="ubuntu"
elif cat /proc/version | grep -q -E -i "centos|red hat|redhat"; then
    release="centos"
else
    OUT_ERROR "[错误] 不支持的操作系统！"
fi

case $(uname -m) in
    aarch64 | arm64) BARCH=arm64 && ARCH=arm64 ;;
    x86_64 | amd64) BARCH=amd64 && ARCH=x86_64 ;;
    *) OUT_ERROR "[错误] 不支持的操作系统！" ;;
esac

cd ~

OUT_ALERT "[提示] 生成密钥中"
SECRET=$(openssl rand -hex 12)

OUT_ALERT "[提示] 下载程序中"
TMP_DIR=$(mktemp -d)
cd $TMP_DIR
STREAM_VER="$(curl -s "https://api.github.com/repos/betaxab/stream/releases" | grep 'tag_name' | head -n 1 | cut -d '"' -f 4)"
wget -O release.zip "https://github.com/betaxab/stream/releases/download/${STREAM_VER}/stream-linux-${BARCH}.zip" || OUT_ERROR "[错误] 下载程序失败！"

OUT_ALERT "[提示] 解压程序中"

unzip release.zip

OUT_ALERT "[提示] 复制程序中"
install stream /usr/bin

OUT_ALERT "[提示] 复制配置中"
cp -f default.json /etc/stream.json
sed -i "s/__API_SECRET__/$SECRET/" /etc/stream.json

OUT_ALERT "[提示] 配置 Logrotate 中"
cat > /etc/logrotate.d/stream << EOF
/var/log/stream.log {
    daily
    rotate 3
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
}
EOF

OUT_ALERT "[提示] 配置服务中"
cat > /etc/systemd/system/stream.service << EOF
[Unit]
Description=Stream Unlock Service
After=network.target

[Service]
Type=simple
LimitNOFILE=infinity
ExecStart=/usr/bin/stream -c /etc/stream.json
Restart=always
RestartSec=4s

[Install]
WantedBy=multi-user.target
EOF

OUT_ALERT "[提示] 重载服务中"
systemctl daemon-reload

OUT_ALERT "[提示] 启动服务中"
systemctl enable stream
systemctl restart stream

OUT_INFO  "[信息] 部署完毕！"
OUT_ALERT "[提示] 您的 DNS 地址 $CURRENT:53"
OUT_ALERT "[提示] 您的 API 密钥 $SECRET"
OUT_ALERT "[提示] 您的 API 地址 http://$CURRENT:8888/aio?secret=$SECRET"
cd ~ && rm -rf $TMP_DIR

exit 0
