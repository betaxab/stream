#!/usr/bin/env bash

# 源代码路径
SRC_PATH="./main/main.go"
# 输出目录
OUT_DIR="./releases"
# 程序名
APP_NAME="stream"
# 禁用 CGO
export CGO_ENABLED=0
# 启用 jsonv2 支持
export GOEXPERIMENT=jsonv2

# 需要编译的目标平台和架构
PLATFORMS=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/386"
  "linux/amd64"
  "linux/arm"
  "linux/arm64"
  "linux/loong64"
  "windows/386"
  "windows/amd64"
  "windows/arm"
  "windows/arm64"
)

# 清理之前的编译产物
rm -rf $OUT_DIR
mkdir -p $OUT_DIR

# 初始化 SHA256SUMS 文件
> "$OUT_DIR/SHA256SUMS.txt"

# 编译并压缩每个平台
for PLATFORM in "${PLATFORMS[@]}"; do
  # 获取平台的 GOOS 和 GOARCH
  GOOS=$(echo $PLATFORM | cut -d'/' -f1)
  GOARCH=$(echo $PLATFORM | cut -d'/' -f2)

  # 设置输出文件路径
  OUT_FILE="$OUT_DIR/$APP_NAME"

  # 设置环境变量并编译
  echo "Building ${GOOS}/${GOARCH}..."
  GOOS=$GOOS GOARCH=$GOARCH go build -trimpath -ldflags="-w -s" -o "$OUT_FILE" $SRC_PATH

  # 判断是否是 Windows 平台，需要添加 .exe 扩展名
  if [ "$GOOS" == "windows" ]; then
    mv "$OUT_FILE" "$OUT_FILE.exe"
    OUT_FILE="$OUT_FILE.exe"
  fi

  # 压缩成 .zip 文件
  echo "ZIP ${OUT_FILE}..."
  zip -j "$OUT_DIR/$APP_NAME-${GOOS}-${GOARCH}.zip" "$OUT_FILE" default.json

  # 计算 SHA256 值并追加到 SHA256SUMS.txt 中
  sha256sum "$OUT_DIR/$APP_NAME-${GOOS}-${GOARCH}.zip" | sed "s@$OUT_DIR/@@g" >> "$OUT_DIR/SHA256SUMS.txt"

  # 清理中间文件
  rm "$OUT_FILE"
done

echo "The build and compression are complete, and all files are stored in the $OUT_DIR directory."

exit 0