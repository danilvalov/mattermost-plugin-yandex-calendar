# Git submodule third_party/go-webdav + local patch (reset tree to tag, then apply; idempotent).
# GIT_CONFIG_*: skip global url.insteadOf that rewrites https://github.com to SSH (CI / sandboxes).
#
# Pinned tag must match go.mod: github.com/emersion/go-webdav v0.7.0 (peeled: refs/tags/v0.7.0^{}).
.PHONY: submodules verify-submodule-pins

verify-submodule-pins:
	@set -e; \
	export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null; \
	peel_go=$$(git ls-remote https://github.com/emersion/go-webdav.git 'refs/tags/v0.7.0^{}' | cut -f1); \
	idx_go=$$(git ls-files -s third_party/go-webdav | awk '{print $$2}'); \
	test -n "$$peel_go" && test -n "$$idx_go"; \
	test "$$peel_go" = "$$idx_go" || { echo >&2 "third_party/go-webdav: index $$idx_go != upstream v0.7.0 peeled $$peel_go"; exit 1; }; \
	echo "Submodule gitlink matches upstream tag v0.7.0 ($$idx_go)."

submodules:
	export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null && \
	git submodule update --init --recursive && \
	test -f third_party/go-webdav/go.mod || ( \
		echo >&2 "error: third_party/go-webdav is missing or empty after \`git submodule update\`."; \
		echo >&2 "Ensure .gitmodules is committed and run: git submodule sync --recursive && git submodule update --init --recursive"; \
		exit 1 ) && \
	( cd third_party/go-webdav && ( git fetch -q --tags origin 2>/dev/null || true ) && \
	  git checkout -f v0.7.0 && git clean -fd && \
	  patch -p1 < ../../patches/go-webdav-v0.7.0-yandex-getetag.patch )
