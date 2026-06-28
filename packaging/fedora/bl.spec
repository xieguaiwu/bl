%global debug_package %{nil}

Name:           bl
Version:        1.6.3
Release:        1%{?dist}
Summary:        Terminal dictionary client with LLM translation support

License:        MIT
URL:            https://github.com/xieguaiwu/bl
Source0:        %{url}/archive/v%{version}.tar.gz#/%{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.22

%description
bl is a terminal-based dictionary client written in Go. Features:
• Web scraping: Youdao (EN⇄ZH), WoerterNet (German)
• Offline SQLite dictionaries with zlib compression
• LLM translation via OpenAI-compatible API (--llm flag)
• OpenRouter / OpenCode Zen / NVIDIA provider support
• Automatic provider fallback on failure
• 3 query modes: auto (offline→cache→online), offline, online
• ANSI color output, interactive REPL, JSON output
• Multi-platform bots: Telegram, DingTalk

%prep
%setup -q -n bl-%{version}

%build
go build -o bl -ldflags="-s -w" .

%install
install -Dm755 bl %{buildroot}%{_bindir}/bl
install -Dm644 LICENSE %{buildroot}%{_defaultlicensedir}/%{name}/LICENSE
install -Dm644 README.md %{buildroot}%{_defaultdocdir}/%{name}/README.md
install -Dm644 README.zh-CN.md %{buildroot}%{_defaultdocdir}/%{name}/README.zh-CN.md
install -Dm644 config.example.json %{buildroot}%{_defaultdocdir}/%{name}/config.example.json

%files
%license LICENSE
%doc README.md README.zh-CN.md config.example.json
%{_bindir}/bl

%changelog
* Thu Jun 25 2026 xgw <xieguaiwu@163.com> - 1.6.3-1
- Add provider/model label on every LLM query
- Compact translation output on single line
- Smart fallback: skip retry on bad user input
- Automatic provider fallback on API failure
- Support --from-lang for ambiguous shared words
- Rich grammar info: gender, plural, comparative, superlative
- LLM translation via OpenAI-compatible API
- .blrc local config file support

* Sun Jun 07 2026 xgw <xieguaiwu@163.com> - 1.0.0-1
- Initial package
