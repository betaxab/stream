#!/usr/bin/env bash
# domainlist_to_smartdns.sh
# Convert DOMAIN/DOMAIN-SUFFIX rules to SmartDNS nameserver lines.
# Usage: ./domainlist_to_smartdns.sh [--no-ipv6] domain.list proxy_group output_name.conf

convert_rules() {
    local infile="$1" group="$2" spreffix="$3" outfile="$4"
    awk -v group="$group" -v spreffix="$spreffix" '
    {
        orig=$0
        line=$0

        # 空行直接输出
        if (line ~ /^[ \t]*$/) { print ""; next }

        commented = 0
        # 处理注释：保留注释标记，但尝试解析规则
        if (line ~ /^[ \t]*#/) {
            commented = 1
            sub(/^[ \t]*#/, "", line)   # 去掉行首的 #
            sub(/^[ \t]+/, "", line)    # 去掉 # 后面的空白
        }

        # 匹配 DOMAIN 或 DOMAIN-SUFFIX 规则
        if (match(line, /^(DOMAIN-SUFFIX|DOMAIN)[ \t]*,[ \t]*([A-Za-z0-9.-]+)/, m)) {
            domain = m[2]
            prefix = commented ? "# " : ""
            print prefix spreffix " /" domain "/" group
            next
        }

        # DOMAIN-KEYWORD 注释掉
        if (match(line, /^DOMAIN-KEYWORD[ \t]*,[ \t]*(.+)/, k)) {
            kw = k[0]
            print "# SMARTDNS_NOT_SUPPORT: " kw
            next
        }

        # 不是识别的规则就原样输出（含注释）
        print orig
    }
    ' "$infile" > "$outfile"
}

no_ipv6=0
if [ "${1:-}" == "--no-ipv6" ]; then
    no_ipv6=1
    shift
fi

if [ $# -lt 1 ]; then
    echo "Usage: $0 [--no-ipv6] input_file [group] [output_file]" >&2
    exit 1
fi

infile="$1"
group="${2:-group}"
if [ -n "$3" ]; then
    outfile="$3"
else
    outfile="output.conf"
fi

echo "Converting \"$infile\" to SmartDNS format with group '$group'..."
preffix="nameserver"
convert_rules "$infile" "$group" "$preffix" "$outfile"
echo "Conversion complete. Output written to $outfile."

# 如需生成 IPv6 禁用文件
if [ "$no_ipv6" == "1" ]; then
    preffix="address"
    group="#6"
    outfile="${outfile%.*}-disableipv6.${outfile##*.}"
    convert_rules "$infile" "$group" "$preffix" "$outfile"
    echo "Conversion complete. Output written to $outfile."
fi

exit 0