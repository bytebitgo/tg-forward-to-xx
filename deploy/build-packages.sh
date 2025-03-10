#!/bin/bash
set -e

# 获取脚本所在目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd ${SCRIPT_DIR}

# 检查依赖
check_dependencies() {
    echo "检查依赖..."
    
    # 检查 Go
    if ! command -v go &> /dev/null; then
        echo "错误: 未找到 Go 编译器，请先安装 Go"
        exit 1
    fi
    
    # 检查 RPM 构建工具
    if [ "$BUILD_RPM" = true ] && ! command -v rpmbuild &> /dev/null; then
        echo "警告: 未找到 rpmbuild，将跳过 RPM 包构建"
        BUILD_RPM=false
    fi
    
    # 检查 DEB 构建工具
    if [ "$BUILD_DEB" = true ] && ! command -v dpkg-deb &> /dev/null; then
        echo "警告: 未找到 dpkg-deb，将跳过 DEB 包构建"
        BUILD_DEB=false
    fi
}

# 更新版本号
update_version() {
    echo "当前版本: $VERSION"
    read -p "是否更新版本号? (y/n): " UPDATE_VERSION
    
    if [ "$UPDATE_VERSION" = "y" ] || [ "$UPDATE_VERSION" = "Y" ]; then
        read -p "输入新版本号 (格式: x.y.z): " NEW_VERSION
        
        # 更新 main.go 中的版本号
        sed -i "s/version *= *\"[0-9]*\.[0-9]*\.[0-9]*\"/version = \"$NEW_VERSION\"/" ../cmd/tgforward/main.go
        
        # 更新 RPM spec 文件中的版本号
        sed -i "s/Version: *[0-9]*\.[0-9]*\.[0-9]*/Version:        $NEW_VERSION/" rpm/tg-forward.spec
        
        # 更新 DEB control 文件中的版本号
        sed -i "s/Version: *[0-9]*\.[0-9]*\.[0-9]*/Version: $NEW_VERSION/" deb/control
        
        # 更新 README.md 中的版本号
        sed -i "s/当前版本：v[0-9]*\.[0-9]*\.[0-9]*/当前版本：v$NEW_VERSION/" ../README.md
        
        echo "版本号已更新为 $NEW_VERSION"
        VERSION=$NEW_VERSION
        
        # 更新 CHANGELOG.md
        DATE=$(date +"%Y-%m-%d")
        echo "请输入更新内容 (输入空行结束):"
        CHANGES=""
        while IFS= read -r line; do
            [ -z "$line" ] && break
            CHANGES="${CHANGES}- ${line}\n"
        done
        
        # 插入新的更新日志
        CHANGELOG_CONTENT="## [$NEW_VERSION] - $DATE\n\n### 改进\n\n${CHANGES}\n$(cat ../CHANGELOG.md | tail -n +3)"
        echo -e "# 更新日志\n\n所有版本的重要更改都将记录在此文件中。\n\n${CHANGELOG_CONTENT}" > ../CHANGELOG.md
        
        echo "CHANGELOG.md 已更新"
    fi
}

# 构建 RPM 包
build_rpm() {
    if [ "$BUILD_RPM" = true ]; then
        echo "构建 RPM 包..."
        cd ${SCRIPT_DIR}/rpm
        bash build-rpm.sh
        echo "RPM 包构建完成"
    else
        echo "跳过 RPM 包构建"
    fi
}

# 构建 DEB 包
build_deb() {
    if [ "$BUILD_DEB" = true ]; then
        echo "构建 DEB 包..."
        cd ${SCRIPT_DIR}/deb
        bash build-deb.sh
        echo "DEB 包构建完成"
    else
        echo "跳过 DEB 包构建"
    fi
}

# 主函数
main() {
    # 获取版本号
    VERSION=$(grep "version" ../cmd/tgforward/main.go | grep -oP '"[0-9]+\.[0-9]+\.[0-9]+"' | tr -d '"')
    
    # 默认构建所有包
    BUILD_RPM=true
    BUILD_DEB=true
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --rpm-only)
                BUILD_DEB=false
                shift
                ;;
            --deb-only)
                BUILD_RPM=false
                shift
                ;;
            --no-version-update)
                NO_VERSION_UPDATE=true
                shift
                ;;
            *)
                echo "未知参数: $1"
                exit 1
                ;;
        esac
    done
    
    # 检查依赖
    check_dependencies
    
    # 更新版本号
    if [ "$NO_VERSION_UPDATE" != true ]; then
        update_version
    fi
    
    # 构建包
    build_rpm
    build_deb
    
    echo "所有包构建完成！"
    echo "包文件位于项目根目录"
}

# 执行主函数
main "$@" 