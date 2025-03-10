Name:           tg-forward
Version:        1.0.4
Release:        1%{?dist}
Summary:        Telegram 转发到钉钉服务

License:        MIT
URL:            https://github.com/user/tg-forward-to-xx
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.18
Requires:       systemd

%description
一个用 Golang 实现的应用程序，用于将 Telegram 群聊消息转发到钉钉机器人。
支持处理网络超时情况，对于网络问题发送失败的消息，会暂时存入队列，
等网络正常后再重新发送，程序重启后信息不会丢失。

%prep
%setup -q

%build
go build -o %{name} cmd/tgforward/main.go

%install
# 创建目录
mkdir -p %{buildroot}/opt/%{name}
mkdir -p %{buildroot}%{_unitdir}
mkdir -p %{buildroot}/etc/%{name}
mkdir -p %{buildroot}/var/lib/%{name}/data

# 安装二进制文件
install -m 755 %{name} %{buildroot}/opt/%{name}/

# 安装配置文件
install -m 644 config/config.yaml %{buildroot}/etc/%{name}/

# 安装 systemd 服务文件
install -m 644 deploy/systemd/%{name}.service %{buildroot}%{_unitdir}/

%pre
# 添加用户和组
getent group tgforward >/dev/null || groupadd -r tgforward
getent passwd tgforward >/dev/null || \
    useradd -r -g tgforward -d /opt/%{name} -s /sbin/nologin \
    -c "Telegram 转发到钉钉服务用户" tgforward
exit 0

%post
# 启用服务
%systemd_post %{name}.service
# 设置目录权限
chown -R tgforward:tgforward /opt/%{name}
chown -R tgforward:tgforward /var/lib/%{name}
chmod 750 /opt/%{name}
chmod 750 /var/lib/%{name}
# 设置配置文件权限
chmod 640 /etc/%{name}/config.yaml
chown root:tgforward /etc/%{name}/config.yaml

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service

%files
%defattr(-,root,root,-)
%attr(755,root,root) /opt/%{name}/%{name}
%attr(644,root,root) %{_unitdir}/%{name}.service
%config(noreplace) %attr(640,root,tgforward) /etc/%{name}/config.yaml
%dir %attr(750,tgforward,tgforward) /var/lib/%{name}
%dir %attr(750,tgforward,tgforward) /var/lib/%{name}/data

%changelog
* Wed Aug 30 2023 Developer <dev@example.com> - 1.0.4-1
- 添加 HTTP 服务暴露队列指标数据
- 提供 /metrics 接口返回 JSON 格式指标
- 提供 /health 接口用于健康检查
- 支持配置 HTTP 服务端口和路径

* Fri Aug 25 2023 Developer <dev@example.com> - 1.0.3-1
- 添加队列指标收集功能
- 支持每分钟统计队列状态
- 支持将指标输出到 JSON 文件
- 为对接 Prometheus 监控做准备

* Tue Aug 15 2023 Developer <dev@example.com> - 1.0.2-1
- 添加 systemd 服务支持
- 添加 SysV init 服务脚本
- 添加 RPM 和 DEB 打包支持
- 优化 Linux 系统部署流程

* Tue Aug 15 2023 Developer <dev@example.com> - 1.0.1-1
- 优化错误处理逻辑
- 改进日志输出格式
- 增强网络超时检测

* Tue Aug 01 2023 Developer <dev@example.com> - 1.0.0-1
- 初始版本发布 