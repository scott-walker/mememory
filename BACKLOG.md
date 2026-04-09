# Backlog

## 1. Tags: определить судьбу

Поле `tags` существует в схеме и API, но не используется — ни `recall`, ни `list` не фильтруют по тегам. Нужно определиться: либо добавить фильтрацию (в `list` и/или `recall`), либо выпилить из API и схемы, чтобы не вводить в заблуждение.

## 2. Bootstrap: MCP-инструмент для программной проверки заполненности

**Что уже сделано:** в выводе `mememory bootstrap` появился блок `## Bootstrap Stats` (см. `internal/bootstrap/format.go` → `renderStats`). Он показывает:
- Project name + source (`.mememory file` / `git` / `cwd basename` / `flag`)
- Loaded: N global + M project memories
- Bootstrap: X / 30_000 tokens (Y% of budget)
- Context: X tokens loaded (Z bytes)
- WARNING при превышении бюджета

Это покрывает первоначальные пункты item'а как **read-once** информацию на старте сессии — пользователь и агент видят её в SessionStart hook output.

**Что осталось сделать:** MCP tool, который возвращает ту же информацию **программно по запросу** в любой момент сессии. Use case: агент перед добавлением новой bootstrap-memory хочет проверить, не уйдёт ли он за бюджет, и сделать это структурировано через JSON-ответ MCP, а не парся markdown из hook'а.

**Предлагаемая форма:** новый MCP tool `bootstrap_stats` (или расширение существующего `stats`). Возвращает структуру:

```json
{
  "project": "plexo",
  "project_source": ".mememory file (/path/to/.mememory)",
  "global_count": 10,
  "project_count": 3,
  "total_bytes": 8205,
  "estimated_tokens": 2344,
  "budget_tokens": 30000,
  "budget_percent": 7.8,
  "over_budget": false
}
```

Реализация — переиспользовать `bootstrap.Format(Context)` или вытащить рендер `renderStats` в чистую функцию, возвращающую структуру вместо markdown. Логика подсчёта уже есть, нужен только новый wrapper и регистрация в `internal/mcp/tools.go`.

## 3. Bootstrap: сжатие контента

Когда bootstrap приближается к token-бюджету (`bootstrap.MaxBootstrapTokens` = 30_000), нужен механизм сжатия — переформулировка bootstrap-memories в более компактную форму без потери смысла. Варианты:
- MCP tool `compact` — вызывается агентом, проходит по bootstrap-memories и предлагает сжатые версии
- Автоматическое предупреждение в Stats блоке при достижении 80% бюджета (сейчас warning срабатывает только при 100%+)
- LLM-powered summarization (агент сам переписывает, но нужен UX для подтверждения)

**Связь с item 2:** триггер "приближение к 80%" должен использовать ту же логику подсчёта, что и MCP tool из item 2. Реализовывать имеет смысл вместе.

## 4. Bootstrap: автоматический recall на стороне hook'а

Сейчас bootstrap загружает только memories с `type=bootstrap`. Остальной project-контекст (facts, decisions, rules без тега bootstrap) приходит только если агент вызовет MCP `recall` на первое сообщение пользователя. Это просьба в текстовой инструкции внутри bootstrap-вывода, а не механизм — агент может её проигнорировать или забыть. Реальный инцидент уже был: агент пропустил обязательный recall и работал без проектного контекста до явного указания пользователя.

**Цель:** убрать зависимость от дисциплины агента. Загружать project-context детерминированно на стороне hook'а, чтобы данные оказались в контексте до того, как агент впервые отвечает.

**Уровни реализации (от простого к умному):**

1. **Тупой list.** На session start фетчить top-N project memories по weight (без semantic search). Минимальные изменения: `/api/memories/` уже умеет фильтровать по scope+project, нужно убрать обязательность `type` (если есть) и добавить sort by weight. Новая секция `## Project Context` в выводе bootstrap, отдельный token budget. Реализуется за день. Даёт 80% пользы.

2. **Semantic auto-recall.** Поле `auto_load.queries` в `.mememory` v2: список тематических запросов, для каждого делается HTTP recall в admin API, результаты дедупаются по ID, сортируются по score. Требует нового endpoint'а `POST /api/recall` в admin API (сейчас recall есть только через MCP). Зависимость от embeddings backend'а в hook'е (latency ~200-500ms на запрос). Контролируется пользователем — явно описывает, что хочет видеть в каждой сессии.

3. **Умный авто-recall** (overengineering, не делать без явного запроса). По git context, по recent activity, по clustering. Магия, непредсказуемо.

**Архитектурный вопрос:** один hook (`mememory bootstrap` делает всё) или два (`mememory bootstrap` + `mememory recall-auto` отдельно). Предпочтение — два, для изоляции рисков (recall зависит от embeddings, bootstrap нет).

**Что нужно решить перед стартом:**
- Есть ли в admin API endpoint для recall, или нужно делать.
- Стратегия дедупа memories между bootstrap и auto-recall (одна и та же memory может матчиться в обе секции).
- Бюджет токенов: отдельный для recall (например, `MaxAutoRecallTokens = 50000`) или единый общий.
- Свежесть: фильтр по date/TTL чтобы не загружать устаревшие правила.

**Связь с item 2:** Stats блок в bootstrap output уже показывает размер/процент/количество как read-once информацию. MCP tool из item 2 даст программный доступ к той же логике подсчёта — auto-recall секция должна тоже репортиться в этот же tool единым форматом.
