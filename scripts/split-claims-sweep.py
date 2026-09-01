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
    ("every-workspace-module", r"every (go )?workspace module|all workspace modules", HISTORY),
    ("release-carries-plugins", r"(release|nightly)s? (ships?|carr(y|ies)|publish(es)?|includ(e|es)|contains?)[^.;]{0,80}plugin binar|plugin binaries (among|in) (the|every) (published|release)", HISTORY + r"|ships? no|never (ship|publish|among)|no plugin binar|not (published|among)"),
]

SENTENCE_SPLIT = re.compile(r"(?<=[.;!?])\s+|\s+—\s+")

def sentence_of(window, pos):
    """The sentence/clause of window that contains character offset pos."""
    start = 0
    for sm in SENTENCE_SPLIT.finditer(window):
        if sm.start() >= pos:
            return window[start:sm.start()]
        start = sm.end()
    return window[start:]

def sentence_exempts(window, pos, exempt):
    return re.search(exempt, sentence_of(window, pos), re.I) is not None

def self_test():
    stale = "The configuration moved to a new section during routine cleanup.\n\nRun `make plugins-portable` to build the portable plugin fleet."
    scoped = "The functional plugins moved to davison/topos-plugins; `make plugins-portable` was removed with them."
    ok = True
    for text, want_flag in ((stale, True), (scoped, False)):
        window = ' '.join(text.split('\n'))
        m = re.search(CLASSES[3][1], window, re.I)
        assert m, "self-test pattern miss"
        flagged_here = not sentence_exempts(window, m.start(), CLASSES[3][2])
        print(f"self-test: {'FLAGGED' if flagged_here else 'exempt '} — {text[:60]!r}… (want {'flag' if want_flag else 'exempt'})")
        ok = ok and (flagged_here == want_flag)
    print("self-test:", "PASS" if ok else "FAIL")
    sys.exit(0 if ok else 2)

if len(sys.argv) > 1 and sys.argv[1] == '--self-test':
    self_test()

flagged = 0
seen = set()
for f in files:
    if f.startswith(EXCLUDE_PREFIX) or f.endswith(EXCLUDE_SUFFIX):
        continue
    try:
        text = pathlib.Path(f).read_text()
    except (UnicodeDecodeError, OSError):
        continue
    lines = text.split('\n')
    for i in range(len(lines)):
        window = ' '.join(re.sub(r'^\s*(#|//|\*|-)?\s?', '', l) for l in lines[i:i + 6])
        for name, pat, exempt in CLASSES:
            m = re.search(pat, window, re.I)
            if not m:
                continue
            key = (f, name, i // 6)
            if key in seen:
                continue
            seen.add(key)
            exempted = sentence_exempts(window, m.start(), exempt)
            snippet = window[max(0, m.start() - 40):m.end() + 60]
            flag = '' if exempted else '  <-- CHECK'
            if not exempted:
                flagged += 1
            print(f"{f}:{i + 1} [{name}]{flag} …{snippet}…")
print(f"flagged: {flagged}")
sys.exit(1 if flagged else 0)
