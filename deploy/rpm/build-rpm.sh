#!/bin/bash
set -e

# 获取版本号
VERSION=$(grep "version" ../../cmd/tgforward/main.go | grep -oP '"[0-9]+\.[0-9]+\.[0-9]+"' | tr -d '"')
NAME="tg-forward"
RELEASE="1"

# 创建临时目录
TEMP_DIR=$(mktemp -d)
BUILD_DIR="${TEMP_DIR}/${NAME}-${VERSION}"
mkdir -p ${BUILD_DIR}

# 复制源代码
cp -r ../../cmd ${BUILD_DIR}/
cp -r ../../config ${BUILD_DIR}/
cp -r ../../internal ${BUILD_DIR}/
cp -r ../../deploy ${BUILD_DIR}/
cp ../../go.mod ${BUILD_DIR}/
cp ../../go.sum ${BUILD_DIR}/

# 创建源码包
cd ${TEMP_DIR}
tar -czf "${NAME}-${VERSION}.tar.gz" "${NAME}-${VERSION}"

# 创建 RPM 构建目录
mkdir -p ~/rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# 复制源码包和 spec 文件
cp "${NAME}-${VERSION}.tar.gz" ~/rpmbuild/SOURCES/
cp "${BUILD_DIR}/deploy/rpm/${NAME}.spec" ~/rpmbuild/SPECS/

# 构建 RPM 包
cd ~/rpmbuild/SPECS
rpmbuild -ba ${NAME}.spec

# 复制生成的 RPM 包到当前目录
find ~/rpmbuild/RPMS -name "${NAME}-${VERSION}-${RELEASE}*.rpm" -exec cp {} ../../ \;

# 清理临时目录
rm -rf ${TEMP_DIR}

echo "RPM 包构建完成！" 