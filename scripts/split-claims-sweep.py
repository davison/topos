#!/usr/bin/env python3
# split-claims-sweep.py — the M1-R2 truth sweep (davison/topos#30):
# every tracked maintained file, including Go/TS source comments, is
# searched for claims that describe the kernel repository as it was
# before the plugin split. Exit 1 when any hit is not already scoped to
# history or topos-plugins by its surrounding text. Frozen archives
# (.planning/), milestone records and generated code are excluded, as
# is .claude/ (project-instruction rationale, era-scoped by its own
# framing). Run from the repository root: python3 scripts/split-claims-sweep.py
import re, sys, pathlib, subprocess
root = pathlib.Path('.')
files = subprocess.run(['git','ls-files'],capture_output=True,text=True).stdout.split()
files = [f for f in files if not f.startswith('.claude/')]  # project-instruction rationale, era-scoped by its own framing
exts = ('.go','.ts','.svelte','.md','.sh','.yml','.yaml','.toml','.proto','.json','Makefile')
skip = ('.planning/','docs/milestones/','web/node_modules','web/static/','kernel/webui/build','sdk/gen/')
# Claim classes the four M1-R2 verdicts named (davison/topos#6, #13,
# #24, #28, #30). Windows are 4 lines joined with comment prefixes
# stripped; a hit whose surrounding 8 lines already scope it to history
# or topos-plugins is listed without the CHECK flag.
pats = [
 ("rebuild-every-plugin", r"rebuilds? (every|all)( the)? (trusted )?plugin binar"),
 ("make-target-carries-fleet", r"make (build|build-portable|dev|plugins)\b[^.;]{0,160}\b(rebuild|build)s? (every|all)\b[^.;]{0,60}plugin"),
 ("install-writes-plugins", r"make install\b[^.;]{0,160}(writes? its plugins|places? (the )?plugin binar|plugin binaries (to|at|under) \$?\(?PREFIX)"),
 ("removed-targets", r"\bmake (signal|test-signal|plugins-portable|install-signal|uninstall-signal|gdrive-external-rehearsal)\b"),
 ("removed-scripts", r"install-signal\.sh|signal-readonly-smoke"),
 ("absent-binaries", r"bin/plugins/topos-plugin-(paperless|proton|signal|silverbullet|whatsapp|filesystem|gdrive)"),
 ("absent-dirs", r"\bplugins/(paperless|proton|signal|silverbullet|whatsapp|filesystem|gdrive)\b"),
 ("counted-plugins", r"\b(five|six|seven|four) (source |functional |real |shipped |first-party )?plugins?\b(?! ?-?(owned|populated)| keys)"),
 ("in-repo-plugins", r"in-repo plugin|in this repository'?s? plugins|this repository'?s (own )?(source )?plugins"),
 ("every-workspace-module", r"every (go )?workspace module|all workspace modules"),
 ("release-carries-plugins", r"(release|nightly)s? (ships?|carr(y|ies)|publish(es)?|includ(e|es)|contains?)[^.;]{0,80}plugin binar|plugin binaries (among|in) (the|every) (published|release)"),
]
ok_ctx = re.compile(r"topos-plugins|moved|departed|no longer|removed|before the split|pre-split|older (kernel )?tags|history|used to|formerly|the split|not among|never (among|published|ship)|absent|gone|left|deleted", re.I)
hits = []
flagged = 0
for f in files:
    if not (f.endswith(exts) or f.endswith('Makefile')) or any(f.startswith(s) for s in skip): continue
    try: text = pathlib.Path(f).read_text()
    except Exception: continue
    lines = text.split('\n')
    # paragraph-join: collapse comment prefixes and join adjacent lines into windows
    for i in range(len(lines)):
        window = ' '.join(re.sub(r'^\s*(#|//|\*|-)?\s?', '', l) for l in lines[i:i+4])
        for name, pat in pats:
            m = re.search(pat, window, re.I)
            if m:
                snippet = window[max(0,m.start()-40):m.end()+60].replace('\n',' ')
                ctx = ' '.join(lines[max(0,i-2):i+6])
                flag = '' if ok_ctx.search(ctx) else '  <-- CHECK'
                hits.append((f, i+1, name, snippet, flag))
seen=set()
for f,l,name,snip,flag in hits:
    key=(f,name,l//6)
    if key in seen: continue
    seen.add(key)
    if flag: flagged += 1
    print(f"{f}:{l} [{name}]{flag} …{snip}…")
print(f"flagged: {flagged}")
sys.exit(1 if flagged else 0)
