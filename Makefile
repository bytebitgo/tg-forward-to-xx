.PHONY: all build clean test rpm deb packages install

# 变量定义
NAME := tg-forward
VERSION := $(shell grep "version" cmd/tgforward/main.go | grep -oP '"[0-9]+\.[0-9]+\.[0-9]+"' | tr -d '"')
BINDIR := bin
BINARY := $(BINDIR)/$(NAME)
INSTALLDIR := /opt/$(NAME)
CONFIGDIR := /etc/$(NAME)
DATADIR := /var/lib/$(NAME)

# 默认目标
all: build

# 创建目录
$(BINDIR):
	mkdir -p $(BINDIR)

# 构建应用
build: $(BINDIR)
	go build -o $(BINARY) cmd/tgforward/main.go
	@echo "构建完成: $(BINARY)"
	@echo "版本: $(VERSION)"

# 运行应用
run: build
	$(BINARY) -config config/config.yaml

# 清理构建产物
clean:
	rm -rf $(BINDIR)
	rm -f $(NAME)_*.deb
	rm -f $(NAME)-*.rpm
	@echo "清理完成"

# 运行测试
test:
	go test -v ./...

# 构建 RPM 包
rpm:
	cd deploy && bash build-packages.sh --rpm-only --no-version-update
	@echo "RPM 包构建完成"

# 构建 DEB 包
deb:
	cd deploy && bash build-packages.sh --deb-only --no-version-update
	@echo "DEB 包构建完成"

# 构建所有包
packages:
	cd deploy && bash build-packages.sh
	@echo "所有包构建完成"

# 安装应用
install: build
	@echo "安装 $(NAME) 到 $(INSTALLDIR)"
	sudo mkdir -p $(INSTALLDIR)
	sudo mkdir -p $(CONFIGDIR)
	sudo mkdir -p $(DATADIR)/data
	sudo cp $(BINARY) $(INSTALLDIR)/
	sudo cp -n config/config.yaml $(CONFIGDIR)/
	sudo cp deploy/systemd/$(NAME).service /lib/systemd/system/
	sudo systemctl daemon-reload
	@echo "安装完成"
	@echo "请编辑配置文件: $(CONFIGDIR)/config.yaml"
	@echo "启动服务: sudo systemctl start $(NAME)"

# 卸载应用
uninstall:
	@echo "卸载 $(NAME)"
	sudo systemctl stop $(NAME) || true
	sudo systemctl disable $(NAME) || true
	sudo rm -f /lib/systemd/system/$(NAME).service
	sudo systemctl daemon-reload
	sudo rm -rf $(INSTALLDIR)
	@echo "卸载完成"
	@echo "配置文件和数据目录未删除，如需删除请手动执行:"
	@echo "sudo rm -rf $(CONFIGDIR)"
	@echo "sudo rm -rf $(DATADIR)"

# 帮助信息
help:
	@echo "可用目标:"
	@echo "  all        - 默认目标，构建应用"
	@echo "  build      - 构建应用"
	@echo "  clean      - 清理构建产物"
	@echo "  test       - 运行测试"
	@echo "  rpm        - 构建 RPM 包"
	@echo "  deb        - 构建 DEB 包"
	@echo "  packages   - 构建所有包"
	@echo "  install    - 安装应用到系统"
	@echo "  uninstall  - 从系统卸载应用"
	@echo "  run        - 构建并运行应用"
	@echo "  help       - 显示帮助信息" 