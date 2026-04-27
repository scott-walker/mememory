// Package system_rules holds the system-managed layer of the pinned-delivery
// payload — meta-rules about how the agent must work with mememory itself,
// plus rotated framing imperatives that wrap the user-managed pinned rules.
//
// These texts are product-level: they are not editable through MCP tools and
// not stored in the database. They evolve with mememory releases. The user
// layer (rules stored in Postgres with delivery=pinned) is rendered inside
// the frame this package provides.
//
// Rotation defends against agent adaptation to a repeated reinjection
// payload. Each call to Select returns a different combination of frame
// openings, closings, and meta-rule formulations selected from the variant
// banks below.
package system_rules

import "math/rand"

// FrameOpenVariants are the imperative phrases that open a pinned-payload
// reinjection. One is chosen per call to Select. They all carry the same
// semantic load: "treat the rules below as a checklist, violation = failure".
var FrameOpenVariants = []string{
	"Перед формированием ответа сверь его со следующими правилами. Нарушение pinned-правила = провал задачи.",
	"Активные правила этой сессии — обязательны к проверке перед каждым ответом:",
	"Чек-лист перед ответом: пройдись по списку правил ниже и проверь применимость каждого пункта к своему ответу.",
	"Правила, нарушение которых = провал задачи. Сверь свой ответ перед отправкой:",
	"Обязательная проверка перед ответом: каждое правило ниже должно быть соблюдено.",
}

// FrameCloseVariants close the payload after the rule list. Their job is to
// re-anchor the imperative after the agent has scanned through the rules.
var FrameCloseVariants = []string{
	"Перед действием: подтверди, что ни одно из правил выше не нарушается.",
	"Если правило выше противоречит твоему плану — план меняется, не правило.",
	"Сверка завершена? Только теперь формируй ответ.",
	"Любое отклонение от правил выше — повод остановиться и спросить, а не действовать.",
	"Правила выше — не пожелания. Проверь себя, прежде чем продолжить.",
}

// MetaRule is a single system-managed rule about how the agent works with
// mememory. Each rule carries multiple semantically equivalent formulations;
// Select picks one per render. The ID is stable across releases and used by
// tests to assert presence; it is never shown to the agent.
type MetaRule struct {
	ID       string
	Variants []string
}

// MetaRules — system-level rules about agent behaviour around the memory
// layer itself. Edit this list with care: every entry will be reinjected
// into every UserPromptSubmit on every project, for every user.
var MetaRules = []MetaRule{
	{
		ID: "memory_source",
		Variants: []string{
			"mememory — единственный источник долговременной памяти. Игнорируй встроенные механизмы Claude Code (auto-memory, MEMORY.md, ~/.claude/*/memory/).",
			"Долговременную память хранит только mememory. Файловые механизмы Claude Code (auto-memory, MEMORY.md, ~/.claude/*/memory/) — игнорируй, не читай и не пиши.",
			"Источник памяти один — MCP-сервер mememory. Любые встроенные файловые хранилища Claude Code не используй ни для чтения, ни для записи.",
		},
	},
	{
		ID: "recall_first_turn",
		Variants: []string{
			"На первом сообщении сессии recall — обязательная первая операция. Без него работа над задачей запрещена.",
			"Первое действие в сессии — recall с запросом по текущему проекту. До этого нельзя ни читать файлы, ни редактировать, ни запускать команды.",
			"Открыли сессию — сразу recall. Никакие другие инструменты до первого recall не используются.",
		},
	},
	{
		ID: "verify_before_assert",
		Variants: []string{
			"Память отражает состояние на момент записи. Если факт из памяти противоречит текущему состоянию кода — доверяй коду, не памяти.",
			"Recall возвращает то, что было правдой когда-то. Прежде чем ссылаться на факт из памяти — проверь его в коде, и если расходится, обнови память.",
			"Память — снимок прошлого. Текущая правда — в файлах и git. При конфликте память пересматривается, а не код.",
		},
	},
	{
		ID: "rule_violation_is_failure",
		Variants: []string{
			"Нарушение pinned-правила = провал задачи, не \"почти получилось\". Без градаций.",
			"Pinned-правило не имеет компромиссных трактовок. Соблюдено — задача в работе. Нарушено — задача провалена.",
			"К pinned-правилам не применимы оттенки \"в основном выполнил\". Это бинарная проверка.",
		},
	},
	{
		ID: "bootstrap_is_reference",
		Variants: []string{
			"Bootstrap, загруженный в начале сессии — рабочий справочник. Сверяй свои предположения с ним.",
			"Информация из bootstrap не декоративная — это контекст пользователя и проекта. Перед допущениями сверься с ней.",
			"Bootstrap — это справочник, который ты обязан учитывать. Не работай по предположениям, когда есть факт в bootstrap.",
		},
	},
}

// SelectedRule is a single MetaRule with one variant chosen for this render.
type SelectedRule struct {
	ID   string
	Text string
}

// Selected is the full set of texts chosen for one rendering of the
// system-layer frame: opener, meta-rules (each with one variant), closer.
type Selected struct {
	FrameOpen  string
	FrameClose string
	MetaRules  []SelectedRule
}

// Select picks one formulation for the opener, the closer, and each meta-rule.
// All choices share a single rand source seeded with `seed` — so the same
// seed produces the exact same Selected every time (used by tests). In
// production the caller passes time.Now().UnixNano() for fresh randomness.
func Select(seed int64) Selected {
	r := rand.New(rand.NewSource(seed))

	sel := Selected{
		FrameOpen:  FrameOpenVariants[r.Intn(len(FrameOpenVariants))],
		FrameClose: FrameCloseVariants[r.Intn(len(FrameCloseVariants))],
		MetaRules:  make([]SelectedRule, len(MetaRules)),
	}

	for i, rule := range MetaRules {
		sel.MetaRules[i] = SelectedRule{
			ID:   rule.ID,
			Text: rule.Variants[r.Intn(len(rule.Variants))],
		}
	}

	return sel
}
