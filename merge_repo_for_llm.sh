cat > merge_repo_for_llm.sh <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

OUT="${1:-repo_merged.md}"
MAX_FILE_SIZE="${MAX_FILE_SIZE:-300000}"   # 单文件最大 300KB，防止日志/锁文件撑爆上下文

rm -f "$OUT"

is_binary() {
  local file="$1"
  if file "$file" | grep -qiE 'binary|executable|image|audio|video|archive|compressed'; then
    return 0
  fi
  return 1
}

should_skip() {
  local file="$1"

  case "$file" in
    .git/*|vendor/*|node_modules/*|dist/*|build/*|bin/*|target/*|tmp/*|.idea/*|.gocache/*|.gomodcache/*|.vscode/*)
      return 0
      ;;

    *.png|*.jpg|*.jpeg|*.gif|*.webp|*.ico|*.svg)
      return 0
      ;;

    *.zip|*.tar|*.gz|*.tgz|*.rar|*.7z|*.xz|*.bz2)
      return 0
      ;;

    *.exe|*.dll|*.so|*.dylib|*.a|*.o|*.class|*.jar|*.war)
      return 0
      ;;

    *.log|*.pid|*.lock)
      return 0
      ;;

    *.pem|*.key|*.crt|*.p12|*.jks)
      return 0
      ;;

    .env|*.env)
      return 0
      ;;
  esac

  return 1
}

lang_for_file() {
  local file="$1"
  case "$file" in
    *.go) echo "go" ;;
    *.md) echo "markdown" ;;
    *.yaml|*.yml) echo "yaml" ;;
    *.json) echo "json" ;;
    *.toml) echo "toml" ;;
    *.ini|*.conf) echo "ini" ;;
    *.sh|*.bash) echo "bash" ;;
    *.ps1) echo "powershell" ;;
    Dockerfile|*/Dockerfile) echo "dockerfile" ;;
    Makefile|*/Makefile) echo "makefile" ;;
    *.sql) echo "sql" ;;
    *.proto) echo "protobuf" ;;
    *.mod|go.mod) echo "go" ;;
    *.sum|go.sum) echo "" ;;
    *.env.example|*.env.sample) echo "bash" ;;
    *) echo "" ;;
  esac
}

write_header() {
  {
    echo "# Repository Merged For LLM"
    echo
    echo "Generated at: $(date '+%Y-%m-%d %H:%M:%S')"
    echo
    echo "Purpose: merge source code, documents, and configuration files into one Markdown file for LLM analysis."
    echo
    echo "---"
    echo
  } >> "$OUT"
}

write_tree() {
  {
    echo "## Repository Tree"
    echo
    echo '```text'
  } >> "$OUT"

  if command -v tree >/dev/null 2>&1; then
    tree -a \
      -I '.git|vendor|node_modules|dist|build|bin|target|tmp|*.png|*.jpg|*.jpeg|*.gif|*.webp|*.zip|*.tar|*.gz|*.tgz|*.exe|*.dll|*.so|*.log' \
      >> "$OUT" || true
  else
    find . \
      -path './.git' -prune -o \
      -path './vendor' -prune -o \
      -path './node_modules' -prune -o \
      -path './dist' -prune -o \
      -path './build' -prune -o \
      -path './bin' -prune -o \
      -type f -print | sed 's#^\./##' | sort >> "$OUT"
  fi

  {
    echo '```'
    echo
    echo "---"
    echo
  } >> "$OUT"
}

get_files() {
  if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git ls-files -co --exclude-standard
  else
    find . -type f | sed 's#^\./##'
  fi
}

write_files() {
  echo "## Files" >> "$OUT"
  echo >> "$OUT"

  while IFS= read -r file; do
    [ -f "$file" ] || continue

    if should_skip "$file"; then
      continue
    fi

    if is_binary "$file"; then
      continue
    fi

    size=$(wc -c < "$file" | tr -d ' ')
    if [ "$size" -gt "$MAX_FILE_SIZE" ]; then
      {
        echo
        echo "### File: $file"
        echo
        echo "> Skipped: file too large (${size} bytes, limit ${MAX_FILE_SIZE} bytes)."
        echo
      } >> "$OUT"
      continue
    fi

    lang=$(lang_for_file "$file")

    {
      echo
      echo "### File: $file"
      echo
      echo "\`\`\`${lang}"
      cat "$file"
      echo
      echo "\`\`\`"
    } >> "$OUT"

  done < <(get_files | sort)
}

write_header
write_tree
write_files

echo "OK: merged repository into $OUT"
echo "Tip: MAX_FILE_SIZE=500000 ./merge_repo_for_llm.sh repo_merged.md"
EOF

chmod +x merge_repo_for_llm.sh