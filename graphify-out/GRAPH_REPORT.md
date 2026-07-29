# Graph Report - .  (2026-07-29)

## Corpus Check
- cluster-only mode — file stats not available

## Summary
- 241 nodes · 505 edges · 25 communities (21 shown, 4 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 50 edges (avg confidence: 0.8)
- Token cost: 741 input · 1,762 output

## Graph Freshness
- Built from commit: `bae8465e`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Offline Dictionary Lookup
- Query and Translation Engine
- Translation Rendering and Errors
- Configuration Management
- German Dictionary Parsing
- Terminal Input Handling
- LLM Translation Source
- Interactive Config UI
- German-English Dictionary Builder
- Bot Platform Support
- Caching System
- Publishing Helper Script
- Feature Modules
- Package Configuration
- Bot Support
- Interactive Config UI
- Offline Dictionaries
- Build Logic

## God Nodes (most connected - your core abstractions)
1. `main()` - 28 edges
2. `TranslationData` - 24 edges
3. `Config` - 19 edges
4. `InteractiveConfigWithCache()` - 14 edges
5. `Rdict` - 14 edges
6. `pickQuery()` - 14 edges
7. `RenderTranslation()` - 13 edges
8. `OfflineDictionary` - 12 edges
9. `llmQuery()` - 12 edges
10. `Cache` - 10 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `NewRdict()`  [INFERRED]
  cmd/dingtalk/main.go → internal/dict/rdict.go
- `main()` --calls--> `RenderTranslation()`  [INFERRED]
  cmd/dingtalk/main.go → internal/render/render.go
- `main()` --calls--> `NewRdict()`  [INFERRED]
  cmd/telegram/main.go → internal/dict/rdict.go
- `main()` --calls--> `RenderTranslation()`  [INFERRED]
  cmd/telegram/main.go → internal/render/render.go
- `clearCacheCmd()` --calls--> `New()`  [INFERRED]
  main.go → internal/cache/cache.go

## Import Cycles
- None detected.

## Communities (25 total, 4 thin omitted)

### Community 0 - "Offline Dictionary Lookup"
Cohesion: 0.10
Nodes (22): inputEntry, OfflineDictionary, TranslationData, YoudaoSource, capitalize(), CompressEntry(), CreateOfflineDict(), decompressEntry() (+14 more)

### Community 1 - "Query and Translation Engine"
Cohesion: 0.13
Nodes (26): runConfig, FetchedResult, Format, Rdict, Translator, Client, NewRdict(), NewRdictWithOffline() (+18 more)

### Community 2 - "Translation Rendering and Errors"
Cohesion: 0.13
Nodes (21): Builder, Example, HttpError, Meaning, NoTranslationResults, OfflineUnavailable, Pronunciation, ToChinese (+13 more)

### Community 3 - "Configuration Management"
Cohesion: 0.26
Nodes (19): LocalRC, Config, LLMConfig, LLMProvider, Mode, ConfigPath(), DefaultConfig(), DefaultLLMConfig() (+11 more)

### Community 4 - "German Dictionary Parsing"
Cohesion: 0.24
Nodes (13): GermanEntry, WoerterNetSource, Document, addEnglishTranslations(), containsIPASymbols(), extractBodyMeta(), extractIPA(), guessWordType() (+5 more)

### Community 5 - "Terminal Input Handling"
Cohesion: 0.29
Nodes (14): File, displayWidthRunes(), getTermios(), isTTY(), makeRaw(), readEscSeq(), ReadLine(), readUTF8() (+6 more)

### Community 6 - "LLM Translation Source"
Cohesion: 0.23
Nodes (7): LLMSource, buildPrompt(), extractJSON(), Client, NewLLMSource(), stripCodeFences(), truncate()

### Community 7 - "Interactive Config UI"
Cohesion: 0.38
Nodes (13): addProvider(), changeProviderModel(), clearCacheFromUI(), deleteProvider(), editProvider(), InteractiveConfig(), InteractiveConfigWithCache(), maskKey() (+5 more)

### Community 8 - "German-English Dictionary Builder"
Cohesion: 0.33
Nodes (11): deEnEntry, downloadDict(), extractDefinitions(), extractGender(), extractWord(), extractWordType(), guessWordType(), isUpper() (+3 more)

### Community 9 - "Bot Platform Support"
Cohesion: 0.23
Nodes (9): extractQuery(), main(), escapeHTML(), main(), DictionarySource, dingtalkRequest, dingtalkResponse, dingtalkText (+1 more)

### Community 10 - "Caching System"
Cohesion: 0.24
Nodes (3): Cache, DB, New()

### Community 11 - "Publishing Helper Script"
Cohesion: 0.22
Nodes (7): { chromium }, CONTENT_DIR, DATA_DIR, fs, path, PLATFORMS, URLS

### Community 12 - "Feature Modules"
Cohesion: 0.33
Nodes (6): Cache Management, Chinese Idiom Dictionary, Grammar Analysis, LLM Translation, Provider Fallback, Translation Picker

### Community 13 - "Package Configuration"
Cohesion: 0.50
Nodes (3): dependencies, playwright, playwright

## Knowledge Gaps
- **19 isolated node(s):** `dingtalkRequest`, `bl`, `Translator`, `playwright`, `{ chromium }` (+14 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **4 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `TranslationData` connect `Offline Dictionary Lookup` to `Query and Translation Engine`, `Translation Rendering and Errors`, `German Dictionary Parsing`, `LLM Translation Source`?**
  _High betweenness centrality (0.173) - this node is a cross-community bridge._
- **Why does `main()` connect `Configuration Management` to `Offline Dictionary Lookup`, `Query and Translation Engine`, `LLM Translation Source`, `Interactive Config UI`, `Bot Platform Support`?**
  _High betweenness centrality (0.145) - this node is a cross-community bridge._
- **Why does `pickQuery()` connect `Query and Translation Engine` to `Translation Rendering and Errors`, `Configuration Management`, `Terminal Input Handling`, `LLM Translation Source`?**
  _High betweenness centrality (0.120) - this node is a cross-community bridge._
- **Are the 11 inferred relationships involving `main()` (e.g. with `Mode` and `ConfigPath()`) actually correct?**
  _`main()` has 11 INFERRED edges - model-reasoned connections that need verification._
- **Are the 3 inferred relationships involving `InteractiveConfigWithCache()` (e.g. with `DefaultLLMConfig()` and `Save()`) actually correct?**
  _`InteractiveConfigWithCache()` has 3 INFERRED edges - model-reasoned connections that need verification._
- **What connects `dingtalkRequest`, `bl`, `Translator` to the rest of the system?**
  _19 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Offline Dictionary Lookup` be split into smaller, more focused modules?**
  _Cohesion score 0.1 - nodes in this community are weakly interconnected._