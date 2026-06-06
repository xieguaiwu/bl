Name:           bl
Version:        1.0.0
Release:        1%{?dist}
Summary:        Terminal dictionary client supporting Youdao and WoerterNet

License:        MIT
URL:            https://github.com/xieguaiwu/bl
Source0:        %{url}/archive/v%{version}.tar.gz#/%{name}-%{version}.tar.gz

BuildRequires:  golang

%description
bl is a terminal-based dictionary client written in Go. It supports Youdao
(EN⇄ZH) and WoerterNet (German) dictionary lookups with a three-tier query
chain: offline dictionary, SQLite cache, and online HTTP fetch. Features
include ANSI color output, interactive REPL mode, JSON output, and
multi-platform bot support (Telegram, DingTalk).

%prep
%setup -q -n bl-%{version}

%build
go build -o bl -ldflags="-s -w" .

%install
install -Dm755 bl %{buildroot}%{_bindir}/bl
install -Dm644 LICENSE %{buildroot}%{_defaultlicensedir}/%{name}/LICENSE
install -Dm644 README.md %{buildroot}%{_defaultdocdir}/%{name}/README.md
install -Dm644 README.zh-CN.md %{buildroot}%{_defaultdocdir}/%{name}/README.zh-CN.md

%files
%license LICENSE
%doc README.md README.zh-CN.md
%{_bindir}/bl

%changelog
* Sun Jun 07 2026 xgw <xieguaiwu@163.com> - 1.0.0-1
- Initial package
