#!/usr/bin/env python3
# split-claims-sweep.py — the M1-R2 truth sweep (davison/topos#30):
# every tracked text file is searched for claims that describe the
# kernel repository as it was before the plugin split. A hit is
# exempted ONLY when the SENTENCE containing the match (within a
# six-line window with comment prefixes stripped, split on sentence
# and clause boundaries) carries scoping language appropriate to its
# claim CLASS — history words for removed targets and absent paths, a
# topos-plugins/history reference for counted or "in-repo" plugins —
# never by words in a neighbouring sentence (PR #31 review round 2's
# adversarial fixture: an unrelated "moved" three lines away must not
# exempt "Run make plugins-portable"). Exit 1 on any unexempted hit;
# `--self-test` proves both halves of that rule on built-in fixtures. Excluded, explicitly: the frozen GSD archive (.planning/), the
# milestone records (docs/milestones/), generated code (sdk/gen/),
# the built SPA and lockfiles, screenshots, and .claude/ (project-
# instruction rationale, era-scoped by its own framing). Run from the
# repository root: python3 scripts/split-claims-sweep.py
import re, sys, subprocess, pathlib

files = subprocess.run(['git', 'ls-files'], capture_output=True, text=True).stdout.split()
EXCLUDE_PREFIX = ('.planning/', 'docs/milestones/', 'sdk/gen/', 'kernel/webui/build/', 'web/static/', 'docs/ss/', '.claude/',
                  'scripts/split-claims-sweep.py')  # the pattern file necessarily spells the claims it hunts
EXCLUDE_SUFFIX = ('.png', '.jpg', '.svg', '.ico', '.woff', '.woff2', '.db', '.sum', 'package-lock.json', '.gitkeep')

HISTORY = r"removed|departed|no longer|before the split|pre-split|plugin split|older (kernel )?tags?|history|used to|formerly|the split|moved|left behind|leaves? with|retired|superseded"
ELSEWHERE = r"topos-plugins|" + HISTORY

# (class, pattern, exemption-pattern-for-the-same-window)
CLASSES = [
    ("rebuild-every-plugin", r"rebuilds? (every|all)( the)? (trusted )?plugin binar", HISTORY),
    ("make-target-carries-fleet", r"make (build|build-portable|dev|plugins)\b[^.;]{0,160}\b(rebuild|build)s? (every|all)\b[^.;]{0,60}plugin", HISTORY),
    ("install-writes-plugins", r"make install\b[^.;]{0,160}(writes? its plugins|places? (the )?plugin binar|plugin binaries (to|at|under) \$?\(?PREFIX)", HISTORY),
    # targets the kernel Makefile no longer has and no sibling has either
    ("removed-targets", r"\bmake (signal|plugins-portable|gdrive-external-rehearsal)\b", HISTORY),
    # targets that left the kernel Makefile but live in topos-plugins now —
    # a citation is fine when the window says whose they are
    ("sibling-targets", r"\bmake (test-signal|install-signal|uninstall-signal|build-signal)\b", ELSEWHERE),
    ("removed-scripts", r"install-signal\.sh|signal-readonly-smoke", HISTORY),
    ("absent-binaries", r"bin/plugins/topos-plugin-(paperless|proton|signal|silverbullet|whatsapp|filesystem|gdrive)", HISTORY),
    ("absent-dirs", r"\bplugins/(paperless|proton|signal|silverbullet|whatsapp|filesystem|gdrive)\b", ELSEWHERE),
    ("counted-plugins", r"\b(five|six|seven|four) (source |functional |real |shipped |first-party )?plugins?\b(?! ?-?(owned|populated)| keys)", ELSEWHERE),
    ("in-repo-plugins", r"in-repo plugin|in this repository'?s? plugins|this repository'?s (own )?(source )?plugins\b(?! set)", ELSEWHERE),
    # ownership: the functional fleet, or a named member of it, claimed as
    # this repository's — the class QA read for meaning and the vocabulary
    # sweep missed (#32)
    ("in-repo-fleet", r"(this repositor(y's|y)|in-repo|in this repository)[^.;:,()]{0,60}\b(paperless|silverbullet|proton|signal|whatsapp|gdrive|google drive|filesystem|fleet|functional plugins?)\b", ELSEWHERE),
    ("every-workspace-module", r"every (go )?workspace module|all workspace modules", HISTORY),
    ("release-carries-plugins", r"(release|nightly)s? (ships?|carr(y|ies)|publish(es)?|includ(e|es)|contains?)[^.;]{0,80}plugin binar|plugin binaries (among|in) (the|every) (published|release)", HISTORY + r"|ships? no|never (ship|publish|among)|no plugin binar|not (published|among)"),
]

# Clause boundaries: sentence punctuation, commas, colons, em dashes and
# parentheses. The exemption vocabulary must sit in the same clause as the
# match — "moved to topos-plugins, `make x` was removed" scopes; "moved
# during cleanup, and `make x` builds the fleet" does not. Conservative by
# design: a true sentence that scopes its claim in a neighbouring clause is
# reworded to scope it in the clause that makes it.
#
# The limit, stated: a clause is the finest unit a regex vocabulary can
# scope to. "The removed section explains how `make x` builds the fleet"
# is exempt because "removed" shares the clause, and no punctuation rule
# can tell it from a scoped claim. Within a clause, the reader is the
# check; the instrument guarantees only that nothing outside the clause
# excuses it.
CLAUSE_SPLIT = re.compile(r"(?<=[.;:!?,])\s+|\s+—\s+|\s*[()]\s*")

def clause_of(window, pos):
    """The clause of window that contains character offset pos."""
    start = 0
    for sm in CLAUSE_SPLIT.finditer(window):
        if sm.start() >= pos:
            return window[start:sm.start()]
        start = sm.end()
    return window[start:]

def clause_exempts(window, pos, exempt):
    return re.search(exempt, clause_of(window, pos), re.I) is not None

def self_test():
    """Reviewer and QA fixtures (davison/topos#31 rounds 2–4, #32): a stale
    claim with history language nearby — a sentence above, or the same
    sentence across a comma, a line break, a colon, a parenthetical or an
    em dash — must flag; a claim scoped in its own clause must not."""
    by_name = {name: (pat, exempt) for name, pat, exempt in CLASSES}
    fixtures = (
        ("removed-targets", True, "The configuration moved to a new section during routine cleanup.\n\nRun `make plugins-portable` to build the portable plugin fleet."),
        ("removed-targets", True, "The configuration moved to a new section, while make plugins-portable builds every portable plugin."),
        ("removed-targets", True, "The configuration moved during routine cleanup,\nand make plugins-portable builds every portable plugin."),
        ("removed-targets", True, "The configuration moved during cleanup: run make plugins-portable to build the portable plugin fleet."),
        ("removed-targets", True, "The configuration moved during cleanup (an unrelated change), and make plugins-portable builds every portable plugin."),
        ("removed-targets", True, "The configuration moved during cleanup — make plugins-portable builds every portable plugin."),
        ("removed-targets", False, "The functional plugins moved to davison/topos-plugins; `make plugins-portable` was removed with them."),
        ("removed-targets", False, "At the split the fleet left this repository, and `make plugins-portable` was removed with it."),
        ("in-repo-fleet", True, "a deployment mixing this repository's own trusted paperless/SilverBullet/Proton/Signal/WhatsApp plugins with a third-party external one is the expected shape"),
        ("in-repo-fleet", False, "this repository's functional plugins moved to topos-plugins at the split"),
    )
    ok = True
    for cls, want_flag, text in fixtures:
        pat, exempt = by_name[cls]
        window = ' '.join(text.split('\n'))
        m = re.search(pat, window, re.I)
        assert m, f"self-test pattern miss: {cls}: {text[:50]!r}"
        flagged_here = not clause_exempts(window, m.start(), exempt)
        print(f"self-test: {'FLAGGED' if flagged_here else 'exempt '} — [{cls}] {text[:60]!r}… (want {'flag' if want_flag else 'exempt'})")
        ok = ok and (flagged_here == want_flag)
    # the trusted-dir sentence is a kernel fact and must not even match
    assert not re.search(by_name['in-repo-fleet'][0], "the directory this repository's own `make build`/`make dev` populates", re.I)
    print("self-test:", "PASS" if ok else "FAIL")
    sys.exit(0 if ok else 2)

if len(sys.argv) > 1 and sys.argv[1] == '--self-test':
    self_test()

flagged = 0
for f in files:
    if f.startswith(EXCLUDE_PREFIX) or f.endswith(EXCLUDE_SUFFIX):
        continue
    try:
        text = pathlib.Path(f).read_text()
    except (UnicodeDecodeError, OSError):
        continue
    lines = [re.sub(r'^\s*(#|//|\*|-)?\s?', '', l) for l in text.split('\n')]
    for i, line in enumerate(lines):
        # A window centred on this line, so a clause that starts above or
        # runs on below is judged whole; each match is judged exactly once,
        # on the line it starts on.
        before = ' '.join(lines[max(0, i - 3):i])
        off = len(before) + 1 if before else 0
        window = (before + ' ' if before else '') + ' '.join(lines[i:i + 4])
        for name, pat, exempt in CLASSES:
            for m in re.finditer(pat, window, re.I):
                if not (off <= m.start() < off + len(line) + 1):
                    continue
                exempted = clause_exempts(window, m.start(), exempt)
                snippet = window[max(0, m.start() - 40):m.end() + 60]
                flag = '' if exempted else '  <-- CHECK'
                if not exempted:
                    flagged += 1
                print(f"{f}:{i + 1} [{name}]{flag} …{snippet}…")
print(f"flagged: {flagged}")
sys.exit(1 if flagged else 0)
