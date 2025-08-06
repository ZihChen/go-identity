#!/bin/bash

# === Config ===
ATLASGO="atlas"

MYSQL_HOST="REDACTED_DB_HOST"
MYSQL_PORT="3306"
MYSQL_ROOT_USER="REDACTED_DB_USER_PROD"
MYSQL_ROOT_PASSWORD="REDACTED_DB_PASSWORD_PROD"
MYSQL_DATABASE="ms_fatidentitycat"
ATLAS_ENV="gorm"

# === Functions ===

init_migration() {
  echo "📂 \033[1;36mInitialize migrations folder and SQL file...\033[0m"
  CMD="$ATLASGO migrate diff init --env \"$ATLAS_ENV\""
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
  echo -e "✅ Initialize successfully\n"
}

apply_migration() {
  echo -e "💻 \033[1;36mApplying migrations to remote DB...\033[0m"
  CMD="$ATLASGO migrate apply --env \"$ATLAS_ENV\""
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
  echo -e "✅ Migration applied successfully\n"
}

dry_run() {
  echo -e "💻 \033[1;36mPreviewing migration changes on remote DB...\033[0m"
  CMD="$ATLASGO schema apply --dry-run --env \"$ATLAS_ENV\""
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
  echo -e "✅ Dry run completed\n"
}

generate_diff() {
  if [ -z "$1" ]; then
    echo -e "\033[1;31mMissing argument. Usage: ./migrate.sh gen <files prefix name>\033[0m"
    exit 1
  fi

  PREFIX_NAME="$1"
  echo -e "💻 \033[1;36mGenerating migration files...\033[0m"

  CMD="$ATLASGO migrate diff \"$PREFIX_NAME\" --env \"$ATLAS_ENV\""
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
}

schema_inspect() {
  CMD="$ATLASGO schema inspect --env \"$ATLAS_ENV\""
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
}

migrate_status() {
  CMD="$ATLASGO migrate status --env \"$ATLAS_ENV\""
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
}

migrate_hash() {
  CMD="$ATLASGO migrate hash --env \"$ATLAS_ENV\""
  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"
  eval $CMD
}

compare_diff() {
  echo -e "💻 \033[1;36mCompare migration files changes between local and remote...\033[0m"

  CMD="$ATLASGO schema diff \
    --from \"mysql://${MYSQL_ROOT_USER}:${MYSQL_ROOT_PASSWORD}@${MYSQL_HOST}:${MYSQL_PORT}/${MYSQL_DATABASE}?tls=true\" \
    --to \"file://migrations\" \
    --env \"$ATLAS_ENV\""

  echo -e "👉 \033[1;33mExecuting:\033[0m $CMD"

  # 執行並根據關鍵字著色輸出
  eval $CMD | while IFS= read -r line; do
    if [[ "$line" == *"DROP TABLE"* ]]; then
      echo -e "\033[1;31m$line\033[0m"
    elif [[ "$line" == *"CREATE TABLE"* ]]; then
      echo -e "\033[1;32m$line\033[0m"
    elif [[ "$line" == *"ALTER TABLE"* ]]; then
      echo -e "\033[1;35m$line\033[0m"
    else
      echo "$line"
    fi
  done
}

migrate_down() {
    local version="$1"
    local cmd=""

    if [[ -z "$version" ]]; then
      cmd="atlas migrate down --env gorm"
    else
      cmd="atlas migrate down --env gorm --to-version $version"
    fi

    echo -e "⚠️ \033[1;31mYou are about to execute a rollback:\033[0m"
    echo "$cmd"
    read -p "Are you sure you want to proceed? [Y/N] " confirm

    case "$confirm" in
      [Yy]* )
        echo "✅ Executing..."
        eval "$cmd"
        ;;
      * )
        echo "❌ Cancelled."
        exit 0
        ;;
    esac
}

# === Command Dispatcher ===

case "$1" in
  init)
    init_migration
    ;;
  apply)
    apply_migration
    ;;
  dry-run)
    dry_run
    ;;
  gen)
    generate_diff "$2"
    ;;
  inspect)
    schema_inspect
    ;;
  status)
    migrate_status
    ;;
  diff)
    compare_diff
    ;;
  rollback)
    migrate_down "$2"
    ;;
  hash)
    migrate_hash
    ;;
  *)
    echo "❗ Usage: bash $0 [init|apply|dry-run|gen|inspect|status|diff|rollback|hash]"
    exit 1
    ;;
esac