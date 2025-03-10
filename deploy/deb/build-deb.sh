#!/bin/bash
set -e

# 获取版本号
VERSION=$(grep "version" ../../cmd/tgforward/main.go | grep -oP '"[0-9]+\.[0-9]+\.[0-9]+"' | tr -d '"')
NAME="tg-forward"
ARCH="amd64"

# 创建临时目录
TEMP_DIR=$(mktemp -d)
PKG_DIR="${TEMP_DIR}/${NAME}_${VERSION}-1_${ARCH}"

# 创建目录结构
mkdir -p ${PKG_DIR}/DEBIAN
mkdir -p ${PKG_DIR}/opt/${NAME}
mkdir -p ${PKG_DIR}/etc/${NAME}
mkdir -p ${PKG_DIR}/lib/systemd/system
mkdir -p ${PKG_DIR}/etc/init.d
mkdir -p ${PKG_DIR}/var/lib/${NAME}/data

# 编译应用
cd ../../
go build -o ${PKG_DIR}/opt/${NAME}/${NAME} cmd/tgforward/main.go

# 复制配置文件
cp config/config.yaml ${PKG_DIR}/etc/${NAME}/

# 复制 systemd 服务文件
cp deploy/systemd/${NAME}.service ${PKG_DIR}/lib/systemd/system/

# 复制 init.d 脚本
cp deploy/init.d/${NAME} ${PKG_DIR}/etc/init.d/
chmod 755 ${PKG_DIR}/etc/init.d/${NAME}

# 复制控制文件
cp deploy/deb/control ${PKG_DIR}/DEBIAN/

# 创建 postinst 脚本
cat > ${PKG_DIR}/DEBIAN/postinst << 'EOF'
#!/bin/sh
set -e

# 添加用户和组
if ! getent group tgforward >/dev/null; then
    addgroup --system tgforward
fi
if ! getent passwd tgforward >/dev/null; then
    adduser --system --ingroup tgforward --home /opt/tg-forward --no-create-home --shell /bin/false tgforward
fi

# 设置权限
chown -R tgforward:tgforward /opt/tg-forward
chown -R tgforward:tgforward /var/lib/tg-forward
chmod 750 /opt/tg-forward
chmod 750 /var/lib/tg-forward
chmod 640 /etc/tg-forward/config.yaml
chown root:tgforward /etc/tg-forward/config.yaml

# 启用服务
if [ -x "/bin/systemctl" ]; then
    systemctl daemon-reload
    systemctl enable tg-forward.service
fi

exit 0
EOF
chmod 755 ${PKG_DIR}/DEBIAN/postinst

# 创建 prerm 脚本
cat > ${PKG_DIR}/DEBIAN/prerm << 'EOF'
#!/bin/sh
set -e

# 停止服务
if [ -x "/bin/systemctl" ]; then
    systemctl stop tg-forward.service || true
    systemctl disable tg-forward.service || true
fi

exit 0
EOF
chmod 755 ${PKG_DIR}/DEBIAN/prerm

# 创建 postrm 脚本
cat > ${PKG_DIR}/DEBIAN/postrm << 'EOF'
#!/bin/sh
set -e

if [ "$1" = "purge" ]; then
    # 删除用户和组
    if getent passwd tgforward >/dev/null; then
        deluser --quiet --system tgforward || true
    fi
    if getent group tgforward >/dev/null; then
        delgroup --quiet --system tgforward || true
    fi
    
    # 删除数据目录
    rm -rf /var/lib/tg-forward
fi

# 重新加载 systemd
if [ -x "/bin/systemctl" ]; then
    systemctl daemon-reload || true
fi

exit 0
EOF
chmod 755 ${PKG_DIR}/DEBIAN/postrm

# 构建 DEB 包
cd ${TEMP_DIR}
dpkg-deb --build ${NAME}_${VERSION}-1_${ARCH}

# 复制生成的 DEB 包到当前目录
cp ${NAME}_${VERSION}-1_${ARCH}.deb ../../

# 清理临时目录
rm -rf ${TEMP_DIR}

echo "DEB 包构建完成！" 